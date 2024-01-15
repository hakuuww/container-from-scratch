package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// os.Args
// 0	  1   2     3...
// docker run <cmd> <arguments>

func setHostname(newHostname string) error {
	cmd := exec.Command("sudo", "scutil", "--set", "HostName", newHostname)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}


func must (e error){
	if e != nil {
		panic(e)
	}
}


func run() {
	fmt.Printf("run pid: %d",os.Getegid())

	args := append([]string{"child"}, os.Args[2:]...)

	cmd := exec.Command("/proc/self/exe", args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	/*
	Cloneflags: This field within SysProcAttr is used to pass flags to the clone system call in Linux. The clone system call is a lower-level, more flexible version of fork, used to create new processes. The flags specify what attributes of the parent process should be shared with the child process.

	syscall.CLONE_NEWUTS: This flag indicates that the new process should have its own UTS namespace. The UTS namespace is primarily used for isolating system identifiers – namely, the hostname and the NIS domain name.

	syscall.CLONE_NEWPID: This flag creates a new PID namespace for the process. In a PID namespace, the process has a unique set of process IDs separate from the host system. The first process in this new namespace will have a PID of 1 and will see itself as the init process.

	syscall.CLONE_NEWNS: This flag is used twice, once in Cloneflags and once in Unsharedflags. It indicates that the process will have its own mount namespace, which provides isolation of the list of mount points seen by the processes in the namespace. Any filesystem mounts or unmounts will be local to this namespace and will not affect the global filesystem mount points.

	Unsharedflags: This field in SysProcAttr is specifically for unsharing namespaces of the calling process that are not going to be shared with the child process. The use of syscall.CLONE_NEWNS here ensures that the mount namespace is unshared.
	*/

	cmd.SysProcAttr = &syscall.SysProcAttr{
		//creating a new hostname namespace for the container, and a new PID namespace
		//so the container and the host both has its own set of pids thats seperate of each other
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unsharedflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())

	newHostname := "my-container" // Replace with the desired hostname
	err := setHostname(newHostname)
	if err != nil {
		fmt.Println("Error setting hostname:", err)
	} else {
		fmt.Println("Hostname successfully changed to", newHostname)
	}

	cmd.Run()
}


//This function implements a basic control group that limits the amount of processes (PIDs) thats allowed to run in the container environment

func cg() {
	cgroups := "/sys/fs/cgroup"
	pids := filepath.Join(cgroups,"pids")

	must(os.Mkdir(filepath.Join(pids, "container "), 0700))
	//Allow a maximum of 20 Processes
	must(ioutil.WriteFile(filepath.Join(pids, "container/pids.max"), []byte("20"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "container/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "container/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}


func child() {
	fmt.Printf("child pid: %d",os.Getegid())

	newHostname := "my-container" // Replace with the desired hostname
	err := setHostname(newHostname)
	if err != nil {
		fmt.Println("Error setting hostname:", err)
	} else {
		fmt.Println("Hostname successfully changed to", newHostname)
	}

	cg()

	//setting the root of the container to be this ubuntu file system
	must(syscall.Chroot("./ubuntu-fs"))
	must(syscall.Chdir("./"))

	//mounting all processes of the container to the proc folder in ubuntu-fs
	must(syscall.Mount("proc", "proc", "proc",0,""))

	cmd := exec.Command(os.Args[2], os.Args[3:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(cmd.Run())
	must(syscall.Unmount("/proc",0))

}


func main () {

	if len(os.Args) <= 1 {
		panic("not enough arguments")
	}

	switch os.Args[1]{
	case "run":
		run()
	default:
		panic("unrecognized command " + os.Args[1])
	}
}