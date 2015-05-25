This is a mini LDP Server in Go.

LDP stands for Linked Data Platform and the W3 spec for it can be found [here]( http://www.w3.org/TR/ldp/)

*Warning*: This is my sandbox project as I learn both Go and LDP. The code in this repo very likely does not follow Go's best practices and it certainly does not conform to the LDP spec.


## Compile and run the server
    go build
    ./ldpserver


## Operations supported
Fetch the root

    curl locahost:9001

POST to the root (the slug is fixed to "blog")

    curl -X POST localhost:9001

Fetch the node created

    curl localhost:9001/blog1

POST a non-RDF to the root

    curl -X POST --header "Link: http://www.w3.org/ns/ldp#NonRDFSource; rel=\"type\"" --data "hello world" localhost:9001

    curl -X POST --header "Link: http://www.w3.org/ns/ldp#NonRDFSource; rel=\"type\"" --binary-data "@filename" localhost:9001

Fetch the non-RDF created

    curl localhost:9001/blog2

HTTP HEAD operations are supported

    curl -I localhost:9001/
    curl -I localhost:9001/blog1
    curl -I localhost:9001/blog2

Add an RDF source to add a child node (you can only add to RDF sources)

    curl -X POST localhost:9001/blog1

See that the child was added

    curl localhost:9001/blog1

Fetch the child

    curl localhost:9001/blog1/blog3


## Storage
Every resource (RDF or non-RDF) is saved in a folder inside the data folder.

Every RDF source is saved on its own folder with single file inside of it. This file is always `meta.rdf` and it has the triples of the node.

Non-RDF are also saved on their own folder and with a `meta.rdf` file for their metadata but also a file `data.bin` with the non-RDF content.

For example, if we have two nodes (blog1 and blog2) and blog1 is an RDF node and blog2 is a non-RDF then the data structure would look as follow:

    /data/meta.rdf          (root node)
    /data/blog1/meta.rdf    (RDF for blog1)
    /data/blog2/meta.rdf    (RDF for blog2)
    /data/blog2/data.bin    (binary for blog2)


## Misc Notes
I am currently using n-triples rather than turtle because n-triples require less parsing (e.g. no prefixes to be aware of). This should eventually be changed to support and default to turtle.

Blank nodes are only accepted in POST and they are immediately converted to an actual node.


## TODO
A lot. 

* Support Direct and Indirect Containers. Currently only Basic Containers are supported.

* Support HTTP PUT, PATCH, and DELETE. 

* Support turtle as the default RDF serialization format.

* Make sure the ntriples pass a minimum validation. For starters take a look at this set: http://www.w3.org/2000/10/rdf-tests/rdfcore/ntriples/test.nt

* Provide a mechanism to fetch the meta data for a non-RDF (e.g. via a query string or an HTTP header parameter)

* Use BagIt file format to store data (http://en.wikipedia.org/wiki/BagIt)

* Make sure the proper links are included in the HTTP response for all kind of resources. 