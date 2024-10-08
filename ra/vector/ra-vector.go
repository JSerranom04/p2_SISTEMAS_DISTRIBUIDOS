/*******************************************************************************
 * File:		ra/vector/ra-vector.go
 * Authors:		Juan José Serrano Mora, 870282; José Miguel Quílez Vergara, 873499
 * Date:		06-10-2024
*******************************************************************************/

package ra_vector

import (
	"fmt"
	"os"
	"practica2/ms"
	"practica2/utils"
	"reflect"
	"sync"

	"github.com/DistributedClocks/GoVector/govec"
)

// VRequest represents a request for critical section access with vector clock
type VRequest struct {
	VClock []int        // Vector clock
	Pid    int          // Process ID
	OpType utils.OpType // Operation type (READ or WRITE)
}

type VReply struct {
	// Empty struct for reply messages
}

// VRASharedDB represents the shared database for the Vectored Ricart-Agrawala algorithm
type VRASharedDB struct {
	SendVClock []int             // Vector clock for sending messages
	RecVClock  []int             // Vector clock for receiving messages
	OutRepCnt  int               // Count of outstanding replies
	ReqCS      bool              // Flag indicating if the process is requesting critical section
	RepDefd    []bool            // Slice to track deferred replies
	ms         *ms.MessageSystem // Message system for communication
	done       chan bool         // Channel to signal completion
	chrep      chan bool         // Channel for receiving replies
	Mutex      sync.Mutex        // Mutex for synchronization
	logger     *govec.GoLog      // Logger for vector clock operations
	OpType     utils.OpType      // Operation type of this process
	reqChan    chan VRequest     // Channel for incoming requests
}

/*
 *	@Pre:		true.
 *	@Post: 		Creates and initializes a new VRASharedDB instance.
 *
 *	@Returns:	A pointer to the newly created VRASharedDB.
 */
func New(me int, usersFile string, logFile string, opType utils.OpType) *VRASharedDB {
	// Define the message types that the algorithm will use
	messageTypes := []ms.Message{VRequest{}, VReply{}, govec.VClockPayload{}}
	msgs := ms.New(me, usersFile, messageTypes)
	goVecConfig := govec.GetDefaultConfig()
	goVecConfig.EncodingStrategy = utils.CustomEncoder
	goVecConfig.DecodingStrategy = utils.CustomDecoder
	ra := &VRASharedDB{
		SendVClock: make([]int, len(msgs.Peers)),
		RecVClock:  make([]int, len(msgs.Peers)),
		OutRepCnt:  0,
		ReqCS:      false,
		RepDefd:    make([]bool, len(msgs.Peers)),
		ms:         &msgs,
		done:       make(chan bool),
		chrep:      make(chan bool),
		Mutex:      sync.Mutex{},
		logger:     govec.InitGoVector(fmt.Sprintf("VRA-%v", me), logFile, goVecConfig),
		OpType:     opType,
		reqChan:    make(chan VRequest /*, utils.MAX_BUFFERED_REQUESTS*/),
	}
	// Launch message reception & processing routines
	go ra.ReceiveAndDecodeMessage()
	go ra.HandleRequest()
	return ra
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Executes the pre-protocol phase of the Generalized Ricart-Agrawala Algorithm with vector clocks.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) PreProtocol() {
	ra.Mutex.Lock()
	ra.ReqCS = true
	ra.updateSendClock()
	ra.SendVClock[ra.ms.Me-1]++
	ra.Mutex.Unlock()
	ra.OutRepCnt = len(ra.ms.Peers) - 1

	// Send requests to all other processes
	for pid := 1; pid <= len(ra.ms.Peers); pid++ {
		if pid != ra.ms.Me {
			msgPayload := VRequest{VClock: ra.SendVClock, Pid: ra.ms.Me, OpType: ra.OpType}
			loggerMsg := fmt.Sprintf("[PID %v] Sending CS request to process %v, payload: %v", ra.ms.Me, pid, msgPayload)
			utils.LogWithColor(utils.Green, loggerMsg)
			ra.sendClockedMessage(msgPayload, loggerMsg, pid)
		}
	}

	// Wait until all replies are received
	for ra.OutRepCnt > 0 {
		<-ra.chrep
		ra.OutRepCnt--
	}
	utils.LogWithColor(utils.BgLBlue, fmt.Sprintf("[PID %v] Entering critical section (op: %v)...", ra.ms.Me, ra.OpType))
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Sends a clocked message to the specified process.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) sendClockedMessage(messagePayload ms.Message, loggerMsg string, pid int) {
	vClockMessage := ra.logger.PrepareSend(loggerMsg, messagePayload, govec.GetDefaultLogOptions())
	ra.ms.Send(pid, vClockMessage)
}

func (ra *VRASharedDB) updateSendClock() {
	for i := range ra.SendVClock {
		// KEY, NEW: if we just copy ra.RecVClock[i] to ra.SendVClock[i],
		// we'll end up omitting our own process component (which might
		// not have been altered in ra.RecVClock)
		ra.SendVClock[i] = max(ra.SendVClock[i], ra.RecVClock[i])
	}
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Updates the received vector clock.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) updateRecVClock(receivedVClock []int) {
	for i := range ra.RecVClock {
		ra.RecVClock[i] = max(ra.RecVClock[i], receivedVClock[i])
	}
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Compares the local SendVClock with the received vector clock.
 *
 *	@Returns:	Two boolean values: the first indicates if the clocks are comparable and not equal,
 *				the second indicates if the local clock is earlier than the received clock.
 */
func (ra *VRASharedDB) hasEarlierOrEqualSendVClock(receivedVClock []int) (bool, bool) {
	weHaveEarlierClock := false
	weHaveLaterClock := false
	for i := range ra.SendVClock {
		if ra.SendVClock[i] < receivedVClock[i] {
			weHaveEarlierClock = true
		} else if ra.SendVClock[i] > receivedVClock[i] {
			weHaveLaterClock = true
		}
	}
	comparableAndNotEqual := !(weHaveEarlierClock == weHaveLaterClock)
	return comparableAndNotEqual, comparableAndNotEqual && weHaveEarlierClock
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Handles incoming requests for critical section access.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) HandleRequest() {
	for {
		req := <-ra.reqChan
		ra.Mutex.Lock()
		ra.updateRecVClock(req.VClock)
		comparableAndNotEqual, weHaveEarlierClock := ra.hasEarlierOrEqualSendVClock(req.VClock)
		defer_it := ra.ReqCS && (weHaveEarlierClock ||
			(!comparableAndNotEqual && req.Pid > ra.ms.Me)) &&
			utils.ExcludeOps(ra.OpType, req.OpType)
		if defer_it {
			ra.RepDefd[req.Pid-1] = true
			ra.Mutex.Unlock()
			utils.LogWithColor(utils.Blue, fmt.Sprintf("[PID %v] Deferred process %v, RepDefd: %v", ra.ms.Me, req.Pid, ra.RepDefd))
		} else {
			ra.Mutex.Unlock()
			loggerMsg := fmt.Sprintf("[PID %v] Sending immediate CS permission to process %v", ra.ms.Me, req.Pid)
			utils.LogWithColor(utils.Yellow, loggerMsg)
			ra.sendClockedMessage(VReply{}, loggerMsg, req.Pid)
		}
	}
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Receives and decodes incoming messages, routing them to appropriate handlers.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) ReceiveAndDecodeMessage() {
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

			loggerMsg = fmt.Sprintf("[PID %v] Received %v: %v, my_send_clock: %v", ra.ms.Me, reflect.TypeOf(payload).String(), payload, ra.SendVClock)
			utils.LogWithColor(utils.Blue, loggerMsg)
			ra.logger.UnpackReceive(loggerMsg, buff, &dummy, govec.GetDefaultLogOptions())

			switch pl := payload.(type) {
			case VRequest:
				ra.reqChan <- pl
			case VReply:
				ra.chrep <- true
			default:
				utils.LogWithColor(utils.Red, fmt.Sprintf("[PID %v] Received message of unrecognized type", ra.ms.Me))
				os.Exit(1)
			}
		}
	}
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized and PreProtocol has been executed.
 *	@Post: 		Executes the post-protocol phase of the Generalized Ricart-Agrawala Algorithm with vector clocks.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) PostProtocol() {
	utils.LogWithColor(utils.BgOrange, fmt.Sprintf("[PID %v] About to exit critical section (op: %v)...", ra.ms.Me, ra.OpType))
	ra.Mutex.Lock()
	defer ra.Mutex.Unlock()
	ra.ReqCS = false
	for pid := range ra.RepDefd {
		if ra.RepDefd[pid] {
			loggerMsg := fmt.Sprintf("[PID %v] Sending deferred CS permission to process %v", ra.ms.Me, pid+1)
			utils.LogWithColor(utils.Orange, loggerMsg)
			ra.sendClockedMessage(VReply{}, loggerMsg, pid+1)
			ra.RepDefd[pid] = false
		}
	}
}

/*
 *	@Pre:		The VRASharedDB instance is properly initialized.
 *	@Post: 		Stops the message system and signals the end of message processing.
 *
 *	@Returns:	None.
 */
func (ra *VRASharedDB) Stop() {
	ra.ms.Stop()
	ra.done <- true
}
