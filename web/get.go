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
	var pref ldp.PreferTriples

	path := safePath(req.URL.Path)

	if includeBody {
		log.Printf("GET request %s", path)
		pref = ldp.PreferTriples{
			Membership:       isPreferMembership(req.Header),
			MinimalContainer: isPreferMinimalContainer(req.Header)}
		node, err = theServer.GetNode(path, pref)
	} else {
		log.Printf("HEAD request %s", path)
		node, err = theServer.GetHead(path)
	}

	if err != nil {
		handleCommonErrors(resp, req, err)
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
	fmt.Fprint(resp, node.ContentPref(pref))
}
