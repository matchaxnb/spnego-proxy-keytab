package spnegoproxy

import (
	"errors"
	"fmt"
	"net/http"
)

type WebHDFSOp string
type WebHDFSVerb byte

const (
	WebHDFSGet WebHDFSVerb = iota
	WebHDFSPost
	WebHDFSPut
	WebHDFSDelete
)

type WebHDFSEvent struct {
	verb WebHDFSVerb
	op   WebHDFSOp
}

var (
	// HTTP GET operations
	WebHDFSGetOpen               = WebHDFSEvent{WebHDFSGet, "OPEN"}
	WebHDFSGetGetFileStatus      = WebHDFSEvent{WebHDFSGet, "GETFILESTATUS"}
	WebHDFSGetListStatus         = WebHDFSEvent{WebHDFSGet, "LISTSTATUS"}
	WebHDFSGetGetContentSummary  = WebHDFSEvent{WebHDFSGet, "GETCONTENTSUMMARY"}
	WebHDFSGetGetFileChecksum    = WebHDFSEvent{WebHDFSGet, "GETFILECHECKSUM"}
	WebHDFSGetGetHomeDirectory   = WebHDFSEvent{WebHDFSGet, "GETHOMEDIRECTORY"}
	WebHDFSGetGetDelegationToken = WebHDFSEvent{WebHDFSGet, "GETDELEGATIONTOKEN"}

	// HTTP PUT operations
	WebHDFSPutCreate                = WebHDFSEvent{WebHDFSPut, "CREATE"}
	WebHDFSPutMkdirs                = WebHDFSEvent{WebHDFSPut, "MKDIRS"}
	WebHDFSPutRename                = WebHDFSEvent{WebHDFSPut, "RENAME"}
	WebHDFSPutSetReplication        = WebHDFSEvent{WebHDFSPut, "SETREPLICATION"}
	WebHDFSPutSetOwner              = WebHDFSEvent{WebHDFSPut, "SETOWNER"}
	WebHDFSPutSetPermission         = WebHDFSEvent{WebHDFSPut, "SETPERMISSION"}
	WebHDFSPutSetTimes              = WebHDFSEvent{WebHDFSPut, "SETTIMES"}
	WebHDFSPutRenewDelegationToken  = WebHDFSEvent{WebHDFSPut, "RENEWDELEGATIONTOKEN"}
	WebHDFSPutCancelDelegationToken = WebHDFSEvent{WebHDFSPut, "CANCELDELEGATIONTOKEN"}

	// HTTP POST operations
	WebHDFSPostAppend = WebHDFSEvent{WebHDFSPost, "APPEND"}

	// HTTP DELETE operations
	WebHDFSDeleteDelete = WebHDFSEvent{WebHDFSDelete, "DELETE"}

	// Wrong operations (noop)
	WebHDFSWrongDelete = WebHDFSEvent{WebHDFSDelete, ""}
	WebHDFSWrongGet    = WebHDFSEvent{WebHDFSGet, ""}
	WebHDFSWrongPut    = WebHDFSEvent{WebHDFSPut, ""}
	WebHDFSWrongPost   = WebHDFSEvent{WebHDFSPost, ""}
)

type WebHDFSEventChannel chan WebHDFSEvent

// process a request
func ProcessWebHDFSRequestQuery(req *http.Request, eventStream WebHDFSEventChannel) error {

	switch verb := req.Method; verb {
	case http.MethodGet:
		return processWebHDFSGetRequest(req, eventStream)
	case http.MethodPost:
		return processWebHDFSPostRequest(req, eventStream)
	case http.MethodPut:
		return processWebHDFSPutRequest(req, eventStream)
	case http.MethodDelete:
		return processWebHDFSDeleteRequest(req, eventStream)
	default:
		err := fmt.Errorf("unhandled WebHDFS HTTP verb %s", verb)
		return err
	}
}

// processWebHDFSGetRequest processes GET requests for WebHDFS
func processWebHDFSGetRequest(req *http.Request, eventStream WebHDFSEventChannel) error {
	q := req.URL.Query()
	op := q.Get("op")
	switch op {
	case "":
		eventStream <- WebHDFSWrongGet
		return errors.New("GET with no op=")
	case "OPEN":
		eventStream <- WebHDFSGetOpen
	case "GETFILESTATUS":
		eventStream <- WebHDFSGetGetFileStatus
	case "LISTSTATUS":
		eventStream <- WebHDFSGetListStatus
	case "GETCONTENTSUMMARY":
		eventStream <- WebHDFSGetGetContentSummary
	case "GETFILECHECKSUM":
		eventStream <- WebHDFSGetGetFileChecksum
	case "GETHOMEDIRECTORY":
		eventStream <- WebHDFSGetGetHomeDirectory
	case "GETDELEGATIONTOKEN":
		eventStream <- WebHDFSGetGetDelegationToken
	default:
		return fmt.Errorf("processWebHDFSGetRequest unhandled operation: %s", op)
	}
	return nil
}

// processWebHDFSPostRequest processes POST requests for WebHDFS
func processWebHDFSPostRequest(req *http.Request, eventStream WebHDFSEventChannel) error {
	q := req.URL.Query()
	op := q.Get("op")
	switch op {
	case "":
		eventStream <- WebHDFSWrongPost
		return errors.New("POST with no op=")
	case "APPEND":
		eventStream <- WebHDFSPostAppend
	default:
		return fmt.Errorf("processWebHDFSPostRequest unhandled operation: %s", op)
	}
	return nil
}

// processWebHDFSPutRequest processes PUT requests for WebHDFS
func processWebHDFSPutRequest(req *http.Request, eventStream WebHDFSEventChannel) error {
	q := req.URL.Query()
	op := q.Get("op")
	switch op {
	case "":
		eventStream <- WebHDFSWrongPut
		return errors.New("PUT with no op=")
	case "CREATE":
		eventStream <- WebHDFSPutCreate
	case "MKDIRS":
		eventStream <- WebHDFSPutMkdirs
	case "RENAME":
		eventStream <- WebHDFSPutRename
	case "SETREPLICATION":
		eventStream <- WebHDFSPutSetReplication
	case "SETOWNER":
		eventStream <- WebHDFSPutSetOwner
	case "SETPERMISSION":
		eventStream <- WebHDFSPutSetPermission
	case "SETTIMES":
		eventStream <- WebHDFSPutSetTimes
	case "RENEWDELEGATIONTOKEN":
		eventStream <- WebHDFSPutRenewDelegationToken
	case "CANCELDELEGATIONTOKEN":
		eventStream <- WebHDFSPutCancelDelegationToken
	default:
		return fmt.Errorf("processWebHDFSPutRequest unhandled operation: %s", op)
	}
	return nil
}

// processWebHDFSDeleteRequest processes DELETE requests for WebHDFS
func processWebHDFSDeleteRequest(req *http.Request, eventStream WebHDFSEventChannel) error {
	q := req.URL.Query()
	op := q.Get("op")
	switch op {
	case "":
		eventStream <- WebHDFSWrongDelete
		return errors.New("DELETE with no op=")
	case "DELETE":
		eventStream <- WebHDFSDeleteDelete
	default:
		return fmt.Errorf("processWebHDFSDeleteRequest unhandled operation: %s", op)
	}
	return nil
}

func ConsumeWebHDFSEventStream(stream WebHDFSEventChannel) {
	for {
		c := <-stream
		switch c {
		// HTTP GET operations
		case WebHDFSGetOpen:
			webHDFSEvents.get_open += 1
		case WebHDFSGetGetFileStatus:
			webHDFSEvents.get_getfilestatus += 1
		case WebHDFSGetListStatus:
			webHDFSEvents.get_liststatus += 1
		case WebHDFSGetGetContentSummary:
			webHDFSEvents.get_getcontentsummary += 1
		case WebHDFSGetGetFileChecksum:
			webHDFSEvents.get_getfilechecksum += 1
		case WebHDFSGetGetHomeDirectory:
			webHDFSEvents.get_gethomedirectory += 1
		case WebHDFSGetGetDelegationToken:
			webHDFSEvents.get_getdelegationtoken += 1

		// HTTP PUT operations
		case WebHDFSPutCreate:
			webHDFSEvents.put_create += 1
		case WebHDFSPutMkdirs:
			webHDFSEvents.put_mkdirs += 1
		case WebHDFSPutRename:
			webHDFSEvents.put_rename += 1
		case WebHDFSPutSetReplication:
			webHDFSEvents.put_setreplication += 1
		case WebHDFSPutSetOwner:
			webHDFSEvents.put_setowner += 1
		case WebHDFSPutSetPermission:
			webHDFSEvents.put_setpermission += 1
		case WebHDFSPutSetTimes:
			webHDFSEvents.put_settimes += 1
		case WebHDFSPutRenewDelegationToken:
			webHDFSEvents.put_renewdelegationtoken += 1
		case WebHDFSPutCancelDelegationToken:
			webHDFSEvents.put_canceldelegationtoken += 1

		// HTTP POST operations
		case WebHDFSPostAppend:
			webHDFSEvents.post_append += 1

		// HTTP DELETE operations
		case WebHDFSDeleteDelete:
			webHDFSEvents.delete_delete += 1

		// Invalid operations
		case WebHDFSWrongGet:
			webHDFSEvents.get_invalid += 1
		case WebHDFSWrongPut:
			webHDFSEvents.put_invalid += 1
		case WebHDFSWrongPost:
			webHDFSEvents.post_invalid += 1
		case WebHDFSWrongDelete:
			webHDFSEvents.delete_invalid += 1

		default:
			// Handle any unexpected events
			logger.Printf("Unknown event received: %v", c)
		}
	}
}
