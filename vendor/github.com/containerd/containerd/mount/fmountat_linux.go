/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package mount

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/containerd/log"
	"golang.org/x/sys/unix"
)

// fMountat performs mount from the provided directory.
func fMountat(dirfd uintptr, source, target, fstype string, flags uintptr, data string) error {
	var (
		sourceP, targetP, fstypeP, dataP *byte
		pid                              uintptr
		err                              error
		errno, status                    syscall.Errno
	)

	sourceP, err = syscall.BytePtrFromString(source)
	if err != nil {
		return err
	}

	targetP, err = syscall.BytePtrFromString(target)
	if err != nil {
		return err
	}

	fstypeP, err = syscall.BytePtrFromString(fstype)
	if err != nil {
		return err
	}

	if data != "" {
		dataP, err = syscall.BytePtrFromString(data)
		if err != nil {
			return err
		}
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var pipefds [2]int
	if err := syscall.Pipe2(pipefds[:], syscall.O_CLOEXEC); err != nil {
		return fmt.Errorf("failed to open pipe: %w", err)
	}

	defer func() {
		// close both ends of the pipe in a deferred function, since open file
		// descriptor table is shared with child
		syscall.Close(pipefds[0])
		syscall.Close(pipefds[1])
	}()

	pid, errno = forkAndMountat(dirfd,
		uintptr(unsafe.Pointer(sourceP)),
		uintptr(unsafe.Pointer(targetP)),
		uintptr(unsafe.Pointer(fstypeP)),
		flags,
		uintptr(unsafe.Pointer(dataP)),
		pipefds[1],
	)

	if errno != 0 {
		return fmt.Errorf("failed to fork thread: %w", errno)
	}

	defer func() {
		_, err := unix.Wait4(int(pid), nil, 0, nil)
		for err == syscall.EINTR {
			_, err = unix.Wait4(int(pid), nil, 0, nil)
		}

		if err != nil {
			log.L.WithError(err).Debugf("failed to find pid=%d process", pid)
		}
	}()

	_, _, errno = syscall.RawSyscall(syscall.SYS_READ,
		uintptr(pipefds[0]),
		uintptr(unsafe.Pointer(&status)),
		unsafe.Sizeof(status))
	if errno != 0 {
		return fmt.Errorf("failed to read pipe: %w", errno)
	}

	if status != 0 {
		return fmt.Errorf("failed to mount: %w", status)
	}

	return nil
}

// forkAndMountat will fork thread, change working dir and mount.
//
// precondition: the runtime OS thread must be locked.
func forkAndMountat(dirfd uintptr, source, target, fstype, flags, data uintptr, pipefd int) (pid uintptr, errno syscall.Errno) {

	// block signal during clone
	beforeFork()

	// the cloned thread shares the open file descriptor, but the thread
	// never be reused by runtime.
	pid, _, errno = syscall.RawSyscall6(syscall.SYS_CLONE, uintptr(syscall.SIGCHLD)|syscall.CLONE_FILES, 0, 0, 0, 0, 0)
	if errno != 0 || pid != 0 {
		// restore all signals
		afterFork()
		return
	}

	// restore all signals
	afterForkInChild()

	// change working dir
	_, _, errno = syscall.RawSyscall(syscall.SYS_FCHDIR, dirfd, 0, 0)
	if errno != 0 {
		goto childerr
	}
	_, _, errno = syscall.RawSyscall6(syscall.SYS_MOUNT, source, target, fstype, flags, data, 0)

childerr:
	_, _, errno = syscall.RawSyscall(syscall.SYS_WRITE, uintptr(pipefd), uintptr(unsafe.Pointer(&errno)), unsafe.Sizeof(errno))
	syscall.RawSyscall(syscall.SYS_EXIT, uintptr(errno), 0, 0)
	panic("unreachable")
}
