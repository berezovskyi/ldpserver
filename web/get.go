package web

import (
	"fmt"
	"ldpserver/ldp"
	"log"
	"net/http"
)

func handleGet(includeBody bool, resp http.ResponseWriter, req *http.Request) {
	var node ldp.Node
	var err error

	logHeaders(req)
	path := safePath(req.URL.Path)

	if includeBody {
		log.Printf("GET request %s", path)
		node, err = theServer.GetNode(path)
	} else {
		log.Printf("HEAD request %s", path)
		node, err = theServer.GetHead(path)
	}

	switch {
	case err == ldp.NodeNotFoundError:
		log.Printf("Not found %s", path)
		http.NotFound(resp, req)
		return
	case err != nil:
		log.Printf("Error %s", err)
		http.Error(resp, "Could not fetch resource", http.StatusInternalServerError)
		return
	}

	if etag := requestIfNoneMatch(req.Header); etag != "" {
		if etag == node.Etag() {
			resp.WriteHeader(http.StatusNotModified)
			return
		}
	}

	if !node.IsRdf() && isNonRdfMetadataOnlyRequest(req) {
		setResponseHeadersMetadataOnly(resp, node)
		fmt.Fprint(resp, node.Metadata())
		return
	}

	setResponseHeaders(resp, node)
	fmt.Fprint(resp, node.Content())
}
