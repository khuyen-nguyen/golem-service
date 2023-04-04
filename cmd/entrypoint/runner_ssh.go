package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func allowedCommandPatterns() []string {
	return []string{
		"bash",
		"sh",
		"python.*",
		"bundle exec rails .*",
		"bundle exec rake .*",
		"npm .*",
		"yarn .*",
	}
}

func isAllowedCommand(command string) bool {
	for _, allowedPattern := range allowedCommandPatterns() {
		match, _ := regexp.MatchString(allowedPattern, command)
		if match { return true }
	}
	return false
}

func runSsh(envVars map[string]string, _ func(int)) error {
	sshCommand := os.Getenv("SSH_ORIGINAL_COMMAND")
	shellWrapper := "/bin/shell-wrapper.sh"
	port := os.Getenv("SSH_PORT")
	homeDir := "/app"
	authorizedPublicKey := os.Getenv("AUTHORIZED_PUBLIC_KEY")
	needUnlockRoot := os.Getenv("UNLOCK_ROOT") == "1"

	if !isAllowedCommand(sshCommand) {
		log.Infoln("Executing command is not allowed.")
		return fmt.Errorf("Executing command is not allowed.")
	}

	if port == "" || authorizedPublicKey == "" {
		log.Infoln("Missing SSH_PORT or AUTHORIZED_PUBLIC_KEY.")
		return fmt.Errorf("Missing SSH_PORT or AUTHORIZED_PUBLIC_KEY.")
	}

	systemEnvs := []string{}
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "TERM=") &&
			!strings.HasPrefix(value, "SHLVL=") &&
			!strings.HasPrefix(value, "HOME=") &&
			!strings.HasPrefix(value, "PWD=") &&
			!strings.HasPrefix(value, "AUTHORIZED_PUBLIC_KEY=") &&
			!strings.HasPrefix(value, "SSH_PORT=") &&
			!strings.HasPrefix(value, "_") {
			key, val, _ := strings.Cut(value, "=")
			systemEnvs = append(systemEnvs, fmt.Sprintf("%s=%s", key, strconv.Quote(val)))
		}
	}

	ioutil.WriteFile(shellWrapper, []byte(sshWrapper(envVars, systemEnvs)), 0755)

	go func(pid int) {
		select {
		case <-time.After(10 * time.Minute):
			_, err := os.Stat("/var/CONNECTED")
			if os.IsNotExist(err) {
				syscall.Kill(pid, syscall.SIGTERM)
			}
		}
	}(os.Getpid())

	file, err := os.OpenFile("/etc/shells",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Debugf("Error while append to /etc/shells: %+v\n", err)
		return err
	}
	file.WriteString("\n" + shellWrapper + "\n")
	file.Close()

	var cmd *exec.Cmd
	cmd = exec.Command("sed", "-r", "s@^(root:.*?:)[^:]+:[^:]+@\\1"+homeDir+":/bin/bash@g", "-i", "/etc/passwd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Debugf("Failed to start appending /etc/passwd: %+v\n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		log.Debugf("Failed to append /etc/passwd: %+v\n", err)
		return err
	}

	if needUnlockRoot {
		cmd = exec.Command("passwd", "-u", "root")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Debugf("Failed to start [passwd -u root]: %+v\n", err)
			return err
		}
		if err := cmd.Wait(); err != nil {
			log.Debugf("Failed to run [passwd -u root]: %+v\n", err)
			return err
		}
	} else {
		log.Debugf("Bypass unlock root")
	}

	_, err = os.Stat("/etc/pam.d/sshd")
	if err != nil {
		if os.IsNotExist(err) {
			cmd = exec.Command("sed",
				"s@session\\s*required\\s*pam_loginuid.so@session optional pam_loginuid.so@g",
				"-i", "/etc/pam.d/sshd")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if sedErr := cmd.Start(); sedErr != nil {
				log.Debugf("Error while start appending [/etc/pam.d/sshd]: %+v\n", sedErr)
			}
			if sedErr := cmd.Wait(); sedErr != nil {
				log.Debugf("Error while append [/etc/pam.d/sshd]: %+v\n", sedErr)
			}
		} else {
			log.Debugf("Error while checking [/etc/pam.d/sshd]: %+v\n", err)
		}
	}

	os.Mkdir("/var/run/sshd", 0600)
	err = os.Mkdir(fmt.Sprintf("%s/.ssh", homeDir), 0700)
	if err != nil {
		log.Debugf("Error while prepare home: %+v\n", err)
	}
	ioutil.WriteFile(fmt.Sprintf("%s/.ssh/authorized_keys", homeDir),
		[]byte(authorizedPublicKey+"\n"),
		0600)

	customSshdConfigTemplate := `
PubkeyAuthentication yes
PasswordAuthentication no
ForceCommand %s
Port %s
MaxSessions 2
`
	customSshdConfig := fmt.Sprintf(customSshdConfigTemplate, shellWrapper, port)
	ioutil.WriteFile("/etc/custom_sshd_config", []byte(customSshdConfig), 0600)

	cmd = exec.Command("ssh-keygen", "-A")
	if err := cmd.Start(); err != nil {
		log.Debugf("Error while start [ssh-keygen -A]: %+v\n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		log.Debugf("Error while run [ssh-keygen -A]: %+v\n", err)
		return err
	}

	cmd = exec.Command("/usr/sbin/sshd", "-D", "-e",
		"-f", "/etc/custom_sshd_config")
	log.Infoln("Starting ssh server")
	if err := cmd.Start(); err != nil {
		log.Debugf("Error while start [sshd]: %+v\n", err)
		return err
	}
	ioutil.WriteFile("/var/golem-sshd.pid", []byte(strconv.Itoa(cmd.Process.Pid)), 0600)
	if cmd.Wait(); err != nil {
		log.Debugf("Error while run [sshd]: %+v\n", err)
		return err
	}

	return nil
}

func sshWrapper(envVars map[string]string, systemEnvs []string) string {
	template := `#!/bin/bash
if [ -f /var/CONNECTED ]; then
  echo "Connection allocated. Please open another connection."
  exit 1
fi
touch /var/CONNECTED

set -a

# System Environment variables
%s
# Custom Environment variables
%s

set +a
set +e
command=${SSH_ORIGINAL_COMMAND}
# -q            for silent "script"'s messages
# -e            returns command exit code to caller
# /proc/1/fd/1  main process' output, to awslog
# --flush       to flush log immediately
script /proc/1/fd/1 -c "${command}" --flush -q -e

set -e
kill -s SIGTERM ` + "`" + `cat /var/golem-sshd.pid` + "`" + `
`
	envString := "\n"
	for key, value := range envVars {
		envString = fmt.Sprintf("%s\n%s=\"%s\"", envString, key, value)
	}
	return fmt.Sprintf(template, strings.Join(systemEnvs, "\n"), envString)
}
