package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
)

func findFreeDisplay() (int, error) {
	for i := 0; i < 100; i++ {
		_, err := os.Stat(fmt.Sprintf("/tmp/.X11-unix/X%d", i))
		if err != nil {
			if os.IsNotExist(err) {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("not free display")
}

func main() {
	userID := os.Getuid()
	groupID := os.Getgid()

	user, err := user.Current()
	if err != nil {
		log.Fatal("unable to get username", err)
	}

	displayNum, err := findFreeDisplay()
	if err != nil {
		log.Fatal("unable to get display num", err)
	}

	xSocket := fmt.Sprintf("/tmp/.X11-unix/X%d", displayNum)

	// set a cache folder
	cacheFolder := fmt.Sprintf("/tmp/guibox_X%d", displayNum)
	os.RemoveAll(cacheFolder)

	err = os.Mkdir(cacheFolder, os.ModePerm)
	if err != nil {
		log.Println("unable to create directory", err)
	}

	xClientCookie := fmt.Sprintf("%s/Xcookie.client", cacheFolder)
	xServerCookie := fmt.Sprintf("%s/Xcookie.server", cacheFolder)
	xInitrc := fmt.Sprintf("%s/xinitrc", cacheFolder)
	etcPasswd := fmt.Sprintf("%s/passwd", cacheFolder)

	passwdContent := fmt.Sprintf("root:x:0:0:root:/root:/bin/sh\n%s:x:%d:%d:%s,,,:/tmp:/bin/sh\n", user.Username, userID, groupID, user.Username)
	err = os.WriteFile(etcPasswd, []byte(passwdContent), 0644)
	if err != nil {
		log.Fatal("unable to write new unprivileged user", err)
	}

	// PODMAN
	podmanCmd := fmt.Sprintf(`
podman run --rm \
	-e DISPLAY=:%d \
	-e XAUTHORITY=/Xcookie \
	-v %s:/Xcookie:ro \
	-v %s:%s:rw \
	--user %d:%d \
	-v %s:/etc/passwd:ro \
	--group-add audio \
	--env HOME=/tmp \
	--cap-drop ALL \
	--read-only --tmpfs /tmp \
	--security-opt no-new-privileges \
	%s
	`, displayNum, xClientCookie, xSocket, xSocket, userID, groupID, etcPasswd, "docker.io/x11docker/xfce")

	windowManager := ":"

	xInitrcContent := fmt.Sprintf(`#! /bin/bash
export DISPLAY=:%d
export XAUTHORITY=%s

echo '$(setxkbmap -display $DISPLAY -print)' | xkbcomp - :%d
:> %s
xauth add :%d . $(mcookie)
Cookie=$(xauth nlist :%d | sed -e 's/^..../ffff/')
echo $Cookie | xauth -f %s nmerge -
cp %s %s
chmod 644 %s
%s & Windowmanagerpid=$!
%s
	`, displayNum, xClientCookie, displayNum, xClientCookie, displayNum, displayNum, xClientCookie, xClientCookie, xServerCookie, xClientCookie, windowManager, podmanCmd)

	err = os.WriteFile(xInitrc, []byte(xInitrcContent), 0755)
	if err != nil {
		log.Fatal("failed to write xinitrc content", err)
	}

	xInitCommand := exec.Command("xinit", xInitrc, "--", "/usr/bin/Xephyr", fmt.Sprintf(":%d", displayNum),
		"-auth", xServerCookie,
		"-extension", "MIT-SHM",
		"-nolisten", "tcp",
		"-screen", "1000x750x24",
		"-retro",
	)
	xInitCommand.Stdout = os.Stdout
	xInitCommand.Stderr = os.Stderr

	fmt.Println("Starting Xinit...")
	if err := xInitCommand.Run(); err != nil {
		log.Fatalf("Failed to run xinit: %v", err)
	}

	os.RemoveAll(cacheFolder)
}

// func startXephyr(displayNum int) (*exec.Cmd, error) {
// 	cmd := exec.Command("Xephyr",
// 		fmt.Sprintf(":%d", displayNum),
// 		"-extension", "MIT-SHM",
// 		"-extension", "XTEST",
// 		// "-nolisten", "tcp",
// 		// "-screen", "1000x750x24",
// 		// "-retro",
// 	)

// 	err := cmd.Start()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return cmd, nil
// }

// func generateXauthCookie(displayNum int) (string, error) {
// 	cookie, err := exec.Command("mcookie").Output()
// 	if err != nil {
// 		return "", err
// 	}

// 	cookieStr := strings.TrimSpace(string(cookie))
// 	xauthCmd := fmt.Sprintf("xauth add :%d . %s", displayNum, cookieStr)

// 	err = exec.Command("sh", "-c", xauthCmd).Run()
// 	if err != nil {
// 		return "", err
// 	}

// 	return cookieStr, nil
// }
