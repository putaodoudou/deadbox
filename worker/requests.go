package worker

import (
	"fmt"
	"log"

	"encoding/json"

	"github.com/fxnn/deadbox/model"
)

const contentTypePlainText = "text/plain"
const contentTypeJson = "application/json"

type requests struct {
	id   model.WorkerId
	drop model.Drop
}

func (r *requests) pollRequests(p *requestProcessors) error {
	qs, err := r.drop.WorkerRequests(r.id)
	if err != nil {
		return fmt.Errorf("drop returned error: %s", err)
	}

	for _, q := range qs {
		// @todo #7 never process a request twice
		log.Printf("received request %s", q.Id)
		if err = r.processRequest(q, p); err != nil {
			r.sendErrorResponse(q, err)
		}
	}

	return nil
}

func (r *requests) processRequest(request model.WorkerRequest, processors *requestProcessors) error {
	if request.ContentType != contentTypeJson {
		return fmt.Errorf("ContentType not understood by this worker: %s", request.ContentType)
	}

	// @todo #7 decrypt requests

	var content map[string]interface{}
	if err := json.Unmarshal(request.Content, &content); err != nil {
		return fmt.Errorf("content could not be unmarshalled: %s", err)
	}

	requestType, ok := content["requestType"].(string)
	if !ok {
		return fmt.Errorf("requestType field of type string expected")
	}

	processor, ok := processors.requestProcessorForType(requestType)
	if !ok {
		return fmt.Errorf("requestType not known: %s", requestType)
	}

	processorContent := processor.EmptyRequestContent()
	if err := json.Unmarshal(request.Content, processorContent); err != nil {
		return fmt.Errorf("content could not be unmarshalled for requestType %s: %s", requestType, err)
	}

	processorResponse := processor.ProcessRequest(processorContent)

	if responseContent, err := json.Marshal(processorResponse); err != nil {
		return fmt.Errorf("response for requestType %s could not be unmarshalled: %s", requestType, err)
	} else {
		r.sendResponse(request, contentTypeJson, responseContent)
	}

	return nil
}

func (r *requests) sendErrorResponse(q model.WorkerRequest, errToSend error) {
	r.sendResponse(q, contentTypePlainText, []byte(errToSend.Error()))
}

func (r *requests) sendResponse(q model.WorkerRequest, contentType string, content []byte) {
	s := &model.WorkerResponse{
		Timeout:     q.Timeout,
		ContentType: contentType,
		Content:     content,
	}
	if err := r.drop.PutWorkerResponse(r.id, q.Id, s); err != nil {
		log.Printf("drop didn't accept my response for request %s: %s", q.Id, err)
	}
}