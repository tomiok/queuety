# Queuty

#### A simple queue, delivering at-least-one done in pure go.

## Example

### Server
The server in independent and stand-alone solution. It could be run as a docker image
or deploy the pre-compiled binary.

[Example usage](/example/simple-server-client/server)

### Client
The client, for this particular case is written in the same package (because if you fork you just need to change one 
repository) and could be used as a library, inside the manager package.

[Example usage](/example/simple-server-client/client)


