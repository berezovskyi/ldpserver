package server

import (
	"errors"
	"fmt"
	"io"
	"ldpserver/ldp"
	"ldpserver/textstore"
	"ldpserver/util"
	// "log"
)

const defaultSlug string = "node"

type Server struct {
	settings ldp.Settings
	minter   chan string
	// this should use an interface to it's not tied to "textStore"
	nextResource chan textstore.Store
}

func NewServer(rootUri string, dataPath string) Server {
	var server Server
	server.settings = ldp.SettingsNew(rootUri, dataPath)
	ldp.CreateRoot(server.settings)
	server.minter = CreateMinter(server.settings.IdFile())
	server.nextResource = make(chan textstore.Store)
	return server
}

func (server Server) GetNode(path string) (ldp.Node, error) {
	return ldp.GetNode(server.settings, path)
}

func (server Server) GetHead(path string) (ldp.Node, error) {
	return ldp.GetHead(server.settings, path)
}

// PUT
func (server Server) ReplaceRdfSource(triples string, parentPath string, slug string, etag string) (ldp.Node, error) {
	var path string
	isRootNode := (parentPath == ".") && (slug == ".")
	if isRootNode {
		path = "/"
	} else {
		newPath, err := server.getNewPath(slug)
		if err != nil {
			return ldp.Node{}, err
		}
		path = util.UriConcat(parentPath, newPath)
	}

	resource := server.createResourceFromPath(path)
	if resource.Error() != nil && resource.Error() != textstore.AlreadyExistsError {
		return ldp.Node{}, resource.Error()
	}

	if resource.Error() == textstore.AlreadyExistsError {
		// call replace existing (must validate etag, validate that previous version was RDF too)
		return ldp.ReplaceRdfNode(server.settings, triples, path, etag)
	}

	// create new node, no need to test etag
	node, err := ldp.NewRdfNode(server.settings, triples, path)
	if err != nil {
		return ldp.Node{}, err
	}

	if !isRootNode {
		container, err := server.getContainer(parentPath)
		if err != nil {
			return ldp.Node{}, err
		}

		if err := container.AddChild(node); err != nil {
			return ldp.Node{}, err
		}
	}

	return node, nil
}

// POST
func (server Server) CreateRdfSource(triples string, parentPath string, slug string) (ldp.Node, error) {
	container, err := server.getContainer(parentPath)
	if err != nil {
		return ldp.Node{}, err
	}

	newPath, err := server.getNewPath(slug)
	if err != nil {
		return ldp.Node{}, err
	}

	// TODO: Allow overwriting of resources on PUT.
	//       Need to figure out the ramifications of overwriting
	//       a container (e.g. what happens to contained objects?)
	//       or overwriting an RDF Source with a Non-RDF source
	//       (or viceversa)

	resource := server.createResource(parentPath, newPath)
	if resource.Error() != nil {

		if resource.Error() != textstore.AlreadyExistsError {
			return ldp.Node{}, resource.Error()
		}

		if slug == "" {
			// We generated a duplicate node.
			return ldp.Node{}, ldp.DuplicateNodeError
		}

		// The user provided slug is duplicated.
		// Let's try with one of our own.
		return server.CreateRdfSource(triples, parentPath, "")
	}

	path := util.UriConcat(parentPath, newPath)
	node, err := ldp.NewRdfNode(server.settings, triples, path)
	if err != nil {
		return ldp.Node{}, err
	}

	if err := container.AddChild(node); err != nil {
		return ldp.Node{}, err
	}
	return node, nil
}

func (server Server) CreateNonRdfSource(reader io.ReadCloser, parentPath string, slug string) (ldp.Node, error) {
	container, err := server.getContainer(parentPath)
	if err != nil {
		return ldp.Node{}, err
	}

	newPath, err := server.getNewPath(slug)
	if err != nil {
		return ldp.Node{}, err
	}

	newResource := true
	resource := server.createResource(parentPath, newPath)
	if resource.Error() != nil {
		if resource.Error() == textstore.AlreadyExistsError {
			node, err := ldp.GetHead(server.settings, newPath)
			if err != nil {
				return ldp.Node{}, errors.New("Cannot validate resource to overwrite")
			} else if node.IsRdf() {
				return ldp.Node{}, errors.New("Cannot overwrite RDF Source with Non-RDF Source")
			}
			newResource = false
		} else {
			return ldp.Node{}, resource.Error()
		}
	}

	node, err := ldp.NewNonRdfNode(server.settings, reader, parentPath, newPath)
	if err != nil {
		return node, err
	}

	if newResource {
		if err := container.AddChild(node); err != nil {
			return node, err
		}
	}

	return node, nil
}

func (server Server) PatchNode(path string, triples string) error {
	node, err := ldp.GetNode(server.settings, path)
	if err != nil {
		return err
	}
	return node.Patch(triples)
}

func (server Server) getNewPath(slug string) (string, error) {
	if slug == "" {
		// Generate a new server URI (e.g. node34)
		return MintNextUri(defaultSlug, server.minter), nil
	}

	if !util.IsValidSlug(slug) {
		errorMsg := fmt.Sprintf("Invalid Slug received (%s). Slug must not include special characters.", slug)
		return "", errors.New(errorMsg)
	}
	return slug, nil
}

func (server Server) createResource(parentPath string, newPath string) textstore.Store {
	path := util.UriConcat(parentPath, newPath)
	return server.createResourceFromPath(path)
}

func (server Server) createResourceFromPath(path string) textstore.Store {
	pathOnDisk := util.PathConcat(server.settings.DataPath(), path)
	// Queue up the creation of a new resource
	go func(pathOnDisk string) {
		server.nextResource <- textstore.CreateStore(pathOnDisk)
	}(pathOnDisk)

	// Wait for the new resource to be available.
	resource := <-server.nextResource
	return resource
}

func (server Server) getContainer(path string) (ldp.Node, error) {

	if isRootPath(path) {
		// Shortcut. We know for sure this is a container
		return ldp.GetHead(server.settings, "/")
	}

	node, err := ldp.GetNode(server.settings, path)
	if err != nil {
		return node, err
	} else if !node.IsBasicContainer() {
		errorMsg := fmt.Sprintf("%s is not a container", path)
		return node, errors.New(errorMsg)
	}
	return node, nil
}

func (server Server) getContainerUri(parentPath string) (string, error) {
	if isRootPath(parentPath) {
		return server.settings.RootUri(), nil
	}

	// Make sure the parent node exists and it's a container
	parentNode, err := ldp.GetNode(server.settings, parentPath)
	if err != nil {
		return "", err
	} else if !parentNode.IsBasicContainer() {
		return "", errors.New("Parent is not a container")
	}
	return parentNode.Uri(), nil
}

func isRootPath(path string) bool {
	return path == "" || path == "/"
}
