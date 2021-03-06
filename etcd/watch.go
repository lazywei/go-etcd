package etcd

import (
	"errors"
)

// Errors introduced by the Watch command.
var (
	ErrWatchStoppedByUser = errors.New("Watch stopped by the user via stop channel")
)

// WatchAll returns the first change under the given prefix since the given index.  To
// watch for the latest change, set waitIndex = 0.
//
// If the prefix points to a directory, any change under it, including all child directories,
// will be returned.
//
// If a receiver channel is given, it will be a long-term watch. Watch will block at the
// channel. And after someone receive the channel, it will go on to watch that prefix.
// If a stop channel is given, client can close long-term watch using the stop channel
func (c *Client) WatchAll(prefix string, waitIndex uint64, receiver chan *Response, stop chan bool) (*Response, error) {
	return c.watch(prefix, waitIndex, true, receiver, stop)
}

// Watch returns the first change to the given key since the given index.  To
// watch for the latest change, set waitIndex = 0.
//
// If a receiver channel is given, it will be a long-term watch. Watch will block at the
// channel. And after someone receive the channel, it will go on to watch that
// prefix.  If a stop channel is given, client can close long-term watch using
// the stop channel
func (c *Client) Watch(key string, waitIndex uint64, receiver chan *Response, stop chan bool) (*Response, error) {
	return c.watch(key, waitIndex, false, receiver, stop)
}

func (c *Client) watch(prefix string, waitIndex uint64, recursive bool, receiver chan *Response, stop chan bool) (*Response, error) {
	logger.Debugf("watch %s [%s]", prefix, c.cluster.Leader)
	if receiver == nil {
		return c.watchOnce(prefix, waitIndex, recursive, stop)
	} else {
		for {
			resp, err := c.watchOnce(prefix, waitIndex, recursive, stop)
			if resp != nil {
				waitIndex = resp.ModifiedIndex
				receiver <- resp
			} else {
				return nil, err
			}
		}
	}

	return nil, nil
}

// helper func
// return when there is change under the given prefix
func (c *Client) watchOnce(key string, waitIndex uint64, recursive bool, stop chan bool) (*Response, error) {

	respChan := make(chan *Response)
	errChan := make(chan error)

	go func() {
		options := options{
			"wait": true,
		}
		if waitIndex > 0 {
			options["waitIndex"] = waitIndex
		}
		if recursive {
			options["recursive"] = true
		}

		resp, err := c.get(key, options)

		if err != nil {
			errChan <- err
		}

		respChan <- resp
	}()

	select {
	case resp := <-respChan:
		return resp, nil
	case err := <-errChan:
		return nil, err
	case <-stop:
		return nil, ErrWatchStoppedByUser
	}
}
