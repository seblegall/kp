package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	container string
	pod string
	namespace string
	dest string
	src string
	verbosity string
)


func main() {

	initConfig()

	files, err := ListFiles(src, dest)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"source": src,
			"destination": dest,
		}).Fatal(err.Error())
	}

	if err := CopyFiles(container, pod, namespace, files); err != nil {
		logrus.WithFields(logrus.Fields{
			"namespace": namespace,
			"pod": pod,
			"container": container,
		}).Fatal(err)
	}

}


//initConfig reads the flags value, setup the log level an pars cmd args
func initConfig() {

	flag.Usage = func() {
		fmt.Printf("Copying files or directories from local file system " +
			"to a running container file system (almost) natively\n\n")
		fmt.Printf("Usage:\n  kp [flags] source destination\n\n")
		flag.PrintDefaults()
	}


	flag.StringVar(&container, "c", "", "container ID" )
	flag.StringVar(&pod, "p", "", "pod name" )
	flag.StringVar(&namespace, "n", "", "kubernetes namespace")
	flag.StringVar(&verbosity, "v", logrus.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	flag.Parse()

	if err := setUpLogs(os.Stdout, verbosity); err != nil {
		logrus.Fatal(err.Error())
	}

	if container == "" && pod == ""{
		logrus.Fatal("Please, provide at least a container ID or a pod name")
	}

	src = filepath.Clean(flag.Arg(0))
	dest = filepath.Clean(flag.Arg(1))

	logrus.Debugf("Copying files from %s to %s", src, dest)

}

//setUpLogs set on writer as output for the logs and set up the level
func setUpLogs(out io.Writer, level string) error {

	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	return nil
}


//listFiles returns a list of file to copy from source to the destination
//defined as a map
func ListFiles(src, dest string) (map[string]string, error) {

	files := make(map[string]string)

	fi, err := os.Lstat(src)
	if err != nil {
		return  nil, err
	}

	//If src is a file, then we return a map with only one element
	if !fi.Mode().IsDir() {
		files[src] = filepath.Join(dest, fi.Name())
		logrus.Debug(src," -> ", filepath.Join(dest, fi.Name()))
		return files, nil
	}

	//If src is a directory, we find all files iteratively
	errDir := filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(src, path)
			if err != nil {
				return err

			}

			destPath := filepath.Join(dest, fi.Name(), relPath)
			files[path] = destPath
			logrus.Debug(path," -> ", destPath)
			return nil
		})

	if err != nil {
		return nil, errDir
	}

	return files, nil
}

//copyFiles copy files from their source to the destination, inside the container | pod
func CopyFiles(container, pod, namespace string, files map[string]string) error {

	cmd := computeCommand(container, pod, namespace)

	reader, writer := io.Pipe()
	cmd.Stdin = reader

	go func() {
		defer writer.Close()

		if err := createMappedTar(writer, "/", files); err != nil {
			logrus.Fatal("Error creating tar archive:", err)
		}
	}()

	logrus.Debugf("Running command: %s", cmd.Args)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command %v : %s", cmd, err.Error())
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return err
	}

	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s", stderr)
	}

	if len(stderr) > 0 {
		logrus.Debugf("Command output: [%s], stderr: %s", stdout, stderr)
	} else {
		logrus.Debugf("Command output: [%s]", stdout)
	}
	return nil
}

//computeCommand returns the right command to execute depending on the context (Docker or Kubernetes)
func computeCommand(container, pod, namespace string) *exec.Cmd {

	//if no pod provided, it has to be a docker run
	if pod == "" {
		return exec.Command("docker", "exec", "-i", container, "tar", "xmf", "-",
			"-C", "/", "--no-same-owner")
	}

	//if both container and namespace are provided
	if container != "" && namespace != "" {
		return exec.Command( "kubectl", "exec", pod, "--namespace", namespace, "-c", container, "-i",
			"--", "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	}

	//if only container is provided
	if container != "" {
		return exec.Command( "kubectl", "exec", pod, "-c", container, "-i",
			"--", "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	}

	//if only namespace is provided
	if namespace != "" {
		return exec.Command( "kubectl", "exec", pod, "--namespace", namespace, "-i",
			"--", "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	}

	//if no container neither namespace are provided
	return exec.Command( "kubectl", "exec", pod, "-i",
		"--", "tar", "xmf", "-", "-C", "/", "--no-same-owner")
}

//createMappedTar creates a tar from the files and write it to the specified writer.
func createMappedTar(w io.Writer, root string, files map[string]string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for src, dst := range files {
		if err := addFileToTar(root, src, dst, tw); err != nil {
			return err
		}
	}

	return nil
}

//addFileToTar adds a file to the specified tar.
func addFileToTar(root string, src string, dst string, tw *tar.Writer) error {
	var (
		absPath string
		err     error
	)

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	if filepath.IsAbs(src) {
		absPath = src
	} else {
		absPath, err = filepath.Abs(src)
		if err != nil {
			return err
		}
	}

	tarPath := dst
	if tarPath == "" {
		tarPath, err = filepath.Rel(absRoot, absPath)
		if err != nil {
			return err
		}
	}
	tarPath = filepath.ToSlash(tarPath)

	fi, err := os.Lstat(absPath)
	if err != nil {
		return err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		tarHeader, err := tar.FileInfoHeader(fi, tarPath)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath

		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
	case mode.IsRegular():
		tarHeader, err := tar.FileInfoHeader(fi, tarPath)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath

		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}

		f, err := os.Open(absPath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("writing real file %s : %s", absPath, err.Error())
		}
	case (mode & os.ModeSymlink) != 0:
		target, err := os.Readlink(absPath)
		if err != nil {
			return err
		}
		if filepath.IsAbs(target) {
			logrus.Warnf("Skipping %s. Only relative symlinks are supported.", absPath)
			return nil
		}

		tarHeader, err := tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		tarHeader.Name = tarPath
		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
	default:
		logrus.Warnf("Adding possibly unsupported file %s of type %s.", absPath, mode)
		// Try to add it anyway?
		tarHeader, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}
	}
	return nil
}