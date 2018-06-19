// Copyright 2016, 2017 Thales e-Security, Inc
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package crypto11

import (
	pkcs11 "github.com/miekg/pkcs11"
	"sync"
)

// Mutex protecting sessionPools
var sessionPoolMutex sync.RWMutex

// Map of slot IDs to session pools
var sessionPools map[uint]chan pkcs11.SessionHandle = map[uint]chan pkcs11.SessionHandle{}
var sessionWidth map[uint]uint = map[uint]uint{}
var sessionLoad  map[uint]uint = map[uint]uint{}

const maxSessionWidth uint = 1024 // could be configurable

// Create a new session for a given slot
func newSession(slot uint) (pkcs11.SessionHandle, error) {
	sessionPoolMutex.RUnlock()
	sessionPoolMutex.Lock()
	defer sessionPoolMutex.RLock()
	defer sessionPoolMutex.Unlock()
	sessionLoad[slot] = sessionLoad[slot] + 1

	if session, err := libHandle.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION); err != nil {
		return 0, err
	} else {
		return session, nil
	}
}

// Run a function with a session
//
// setupSessions must have been called for the slot already, otherwise
// there will be a panic.
func withSession(slot uint, f func(session pkcs11.SessionHandle) error) error {
	sessionPoolMutex.RLock()
	defer sessionPoolMutex.RUnlock()

	var session pkcs11.SessionHandle
	var err error

	sessionPool := sessionPools[slot]
	if sessionLoad[slot] < sessionWidth[slot] {
		if session, err = newSession(slot); err != nil {
			return err
		}
	} else {
		session = <-sessionPool
	}
	defer func() {
		// TODO better would be to close the session if the pool is full
		sessionPool <- session
	}()
	return f(session)
}

// Create the session pool for a given slot if it does not exist
// already.
func setupSessions(slot uint, width int) error {
	sessionPoolMutex.Lock()
	defer sessionPoolMutex.Unlock()

	if width <= 0 {
		width = int(maxSessionWidth)
	}
	if _, ok := sessionPools[slot]; !ok {
		sessionPools[slot] = make(chan pkcs11.SessionHandle, width)
		sessionWidth[slot] = uint(width)
		sessionLoad[slot] = 0
	}
	return nil
}
