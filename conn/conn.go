package conn

import (
	"net/rpc"
)

type Conn struct {
	addr string
	conn *rpc.Client
	reconnect bool
}

func NewConn(addr string) (*Conn, error) {
	conn := Conn{
		addr: addr,
		reconnect: false,
	}

	c, err := rpc.Dial("tcp", addr)
	if err != nil {
		return &conn, err
	}

	conn.conn = c
	conn.reconnect = true

	return &conn, nil
}

func (c *Conn) Dial() error {
	conn, err := rpc.Dial("tcp", c.addr)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *Conn) Call(serviceMethod string, args interface{}, reply interface{}) error {
	if c.conn == nil {
		if !c.reconnect {
			return nil
		} else {
			err := c.Dial()
			if err != nil {
				return err
			}
		}
	}

	err := c.conn.Call(serviceMethod, args, reply)
	if err != nil {
		c.conn = nil
	}

	return err
}

