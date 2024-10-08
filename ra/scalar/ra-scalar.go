/*******************************************************************************
 * File:		ra/scalar/ra-scalar.go
 * Authors:		Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:		06-10-2024
*******************************************************************************/

package ra_scalar

import (
	"fmt"
	"log"
	"os"
	"practica2/ms"
	"practica2/utils"
	"reflect"
	"sync"

	"github.com/DistributedClocks/GoVector/govec"
)

// Request represents a request for critical section access
type Request struct {
	Clock  int          // Scalar clock value
	Pid    int          // Process ID
	OpType utils.OpType // Operation type (READ or WRITE)
}

type Reply struct {
	// Empty struct for reply messages
}

// RASharedDB represents the shared database for the Ricart-Agrawala algorithm
type RASharedDB struct {
	OurSeqNum int               // Our sequence number
	HigSeqNum int               // Highest sequence number seen
	OutRepCnt int               // Count of outstanding replies
	ReqCS     bool              // Flag indicating if the process is requesting critical section
	RepDefd   []bool            // Slice to track deferred replies
	ms        *ms.MessageSystem // Message system for communication
	done      chan bool         // Channel to signal completion
	chrep     chan bool         // Channel for receiving replies
	Mutex     sync.Mutex        // Mutex for synchronization
	logger    *govec.GoLog      // Logger for vector clock operations
	OpType    utils.OpType      // Operation type of this process
	reqChan   chan Request      // Channel for incoming requests
}

/*
 *	@Pre:		true.
 *	@Post: 		Creates and initializes a new RASharedDB instance.
 *
 *	@Returns:	A pointer to the newly created RASharedDB.
 */
func New(me int, usersFile string, logFile string, opType utils.OpType) *RASharedDB {
	// Define the message types that the algorithm will use
	messageTypes := []ms.Message{Request{}, Reply{}, govec.VClockPayload{}}
	msgs := ms.New(me, usersFile, messageTypes)
	goVecConfig := govec.GetDefaultConfig()
	goVecConfig.EncodingStrategy = utils.CustomEncoder
	goVecConfig.DecodingStrategy = utils.CustomDecoder
	ra := &RASharedDB{
		OurSeqNum: 0,
		HigSeqNum: 0,
		OutRepCnt: 0,
		ReqCS:     false,
		RepDefd:   make([]bool, len(msgs.Peers)),
		ms:        &msgs,
		done:      make(chan bool),
		chrep:     make(chan bool),
		Mutex:     sync.Mutex{},
		logger:    govec.InitGoVector(fmt.Sprintf("SRA-%v", me), logFile, goVecConfig),
		OpType:    opType,
		reqChan:   make(chan Request, utils.MAX_BUFFERED_REQUESTS),
	}
	// Launch message reception & processing routines
	go ra.ReceiveAndDecodeMessage()
	go ra.HandleRequest()
	return ra
}

/*
 *	@Pre:		The RASharedDB instance is properly initialized.
 *	@Post: 		Executes the pre-protocol phase of the Generalized Ricart-Agrawala Algorithm.
 *
 *	@Returns:	None.
 */
func (ra *RASharedDB) PreProtocol() {
	ra.Mutex.Lock()
	ra.ReqCS = true
	ra.OurSeqNum = ra.HigSeqNum + 1
	ra.Mutex.Unlock()
	ra.OutRepCnt = len(ra.ms.Peers) - 1

	// Send requests to all other processes
	for pid := 1; pid <= len(ra.ms.Peers); pid++ {
		if pid != ra.ms.Me {
			msgPayload := Request{Clock: ra.OurSeqNum, Pid: ra.ms.Me, OpType: ra.OpType}
			loggerMsg := fmt.Sprintf("[PID %v] Sending CS request to process %v, payload: %v", ra.ms.Me, pid, msgPayload)
			utils.LogWithColor(utils.Green, loggerMsg)
			ra.sendClockedMessage(msgPayload, loggerMsg, pid)
		}
	}

	// Wait until all replies are received
	for ra.OutRepCnt > 0 {
		<-ra.chrep
		ra.OutRepCnt-- // NOTE: no mutex acquisition required, as `ra.OutRepCnt` is only accessed by one thread
	}
	utils.LogWithColor(utils.BgLBlue, fmt.Sprintf("[PID %v] Entering critical section (op: %v)...", ra.ms.Me, ra.OpType))
}

/*
 *	@Pre:		The RASharedDB instance is properly initialized.
 *	@Post: 		Sends a clocked message to the specified process.
 *
 *	@Returns:	None.
 */
func (ra *RASharedDB) sendClockedMessage(messagePayload ms.Message, loggerMsg string, pid int) {
	vClockMessage := ra.logger.PrepareSend(loggerMsg, messagePayload, govec.GetDefaultLogOptions())
	ra.ms.Send(pid, vClockMessage)
}

/*
 *	@Pre:		The RASharedDB instance is properly initialized.
 *	@Post: 		Handles incoming requests for critical section access.
 *
 *	@Returns:	None.
 */
func (ra *RASharedDB) HandleRequest() {
	for {
		req := <-ra.reqChan
		ra.Mutex.Lock()
		ra.HigSeqNum = max(ra.HigSeqNum, req.Clock)
		// Original version, with scalar clocks
		log.Printf("Req: %v, req.Clock: %v, req.Pid: %v, req.OpType: %v\n", req, req.Clock, req.Pid, req.OpType)
		defer_it := ra.ReqCS && ((req.Clock > ra.OurSeqNum) ||
			(req.Clock == ra.OurSeqNum && req.Pid > ra.ms.Me)) &&
			utils.ExcludeOps(ra.OpType, req.OpType)
		if defer_it {
			ra.RepDefd[req.Pid-1] = true
			// DONE: unlock mutex here
			ra.Mutex.Unlock()
			utils.LogWithColor(utils.Blue, fmt.Sprintf("[PID %v] Deferred process %v, RepDefd: %v", ra.ms.Me, req.Pid, ra.RepDefd))
		} else {
			// DONE: unlock mutex here
			ra.Mutex.Unlock()
			loggerMsg := fmt.Sprintf("[PID %v] Sending immediate CS permission to process %v", ra.ms.Me, req.Pid)
			utils.LogWithColor(utils.Yellow, loggerMsg)
			ra.sendClockedMessage(Reply{}, loggerMsg, req.Pid)
		}
	}
}

/*
 *	@Pre:		The RASharedDB instance is properly initialized.
 *	@Post: 		Receives and decodes incoming messages, routing them to appropriate handlers.
 *
 *	@Returns:	None.
 */
func (ra *RASharedDB) ReceiveAndDecodeMessage() {
	var clockRepl govec.VClockPayload
	var dummy ms.Message
	var loggerMsg string

	for {
		select {
		case <-ra.done:
			return
		default:
			buff := ra.ms.Receive()
			decErr := utils.CustomDecoder(buff, &clockRepl)
			if decErr != nil {
				utils.LogWithColor(utils.Red, fmt.Sprintf("[PID %v] Error decoding message: %v", ra.ms.Me, decErr))
				os.Exit(1)
			}
			payload := clockRepl.Payload

			loggerMsg = fmt.Sprintf("[PID %v] Received %v: %v, my_send_clock: %v", ra.ms.Me, reflect.TypeOf(payload).String(), payload, ra.OurSeqNum)
			utils.LogWithColor(utils.Blue, loggerMsg)
			ra.logger.UnpackReceive(loggerMsg, buff, &dummy, govec.GetDefaultLogOptions())

			switch pl := payload.(type) {
			case Request:
				ra.reqChan <- pl
			case Reply:
				ra.chrep <- true
			default:
				utils.LogWithColor(utils.Red, fmt.Sprintf("[PID %v] Received message of unrecognized type", ra.ms.Me))
				os.Exit(1)
			}
		}
	}
}

/*
 *	@Pre:		The RASharedDB instance is properly initialized and PreProtocol has been executed.
 *	@Post: 		Executes the post-protocol phase of the Generalized Ricart-Agrawala Algorithm.
 *
 *	@Returns:	None.
 */
func (ra *RASharedDB) PostProtocol() {
	utils.LogWithColor(utils.BgOrange, fmt.Sprintf("[PID %v] About to exit critical section (op: %v)...", ra.ms.Me, ra.OpType))
	ra.Mutex.Lock()
	defer ra.Mutex.Unlock()
	ra.ReqCS = false
	for pid := range ra.RepDefd {
		if ra.RepDefd[pid] {
			loggerMsg := fmt.Sprintf("Process %v sending deferred CS permission", ra.ms.Me)
			ra.sendClockedMessage(Reply{}, loggerMsg, pid+1)
			ra.RepDefd[pid] = false
		}
	}
}

/*
 *	@Pre:		The RASharedDB instance is properly initialized.
 *	@Post: 		Stops the message system and signals the end of message processing.
 *
 *	@Returns:	None.
 */
func (ra *RASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}
