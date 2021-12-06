package sockit

// Authenticator authenticate a Conn.
// if the Conn is acceptable, then an User will Valid()==true should be returned.
type Authenticator interface {
	Auth(c Conn) (User, error)
}

// User is a client identifier.
// if Valid returns false, the connection will be closed.
type User interface {
	Valid() bool
	Id() string
}
