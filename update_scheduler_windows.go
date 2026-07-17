//go:build windows

package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const updateTaskName = `\HermesDock\AutoUpdate`

func (a *App) registerUpdateTask() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("读取当前 Windows 用户失败：%w", err)
	}
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>Hermes Dock automatic update</Description><URI>%s</URI></RegistrationInfo>
  <Triggers><CalendarTrigger><StartBoundary>2000-01-01T02:00:00</StartBoundary><Enabled>true</Enabled><ScheduleByDay><DaysInterval>1</DaysInterval></ScheduleByDay><RandomDelay>PT30M</RandomDelay></CalendarTrigger></Triggers>
  <Principals><Principal id="Author"><UserId>%s</UserId><LogonType>InteractiveToken</LogonType><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><StartWhenAvailable>true</StartWhenAvailable><ExecutionTimeLimit>PT2H</ExecutionTimeLimit><Enabled>true</Enabled></Settings>
  <Actions Context="Author"><Exec><Command>%s</Command><Arguments>--scheduled-update --instance-root &quot;%s&quot;</Arguments></Exec></Actions>
</Task>`, xmlEscape(updateTaskName), xmlEscape(currentUser.Username), xmlEscape(executable), xmlEscape(a.instanceRoot))
	if err := os.MkdirAll(a.updateDir(), 0700); err != nil {
		return err
	}
	xmlPath := filepath.Join(a.updateDir(), "scheduled-task.xml")
	if err := atomicWriteFile(xmlPath, encodeUTF16LE(xml), 0600); err != nil {
		return err
	}
	defer os.Remove(xmlPath)
	output, err := backgroundCommand("schtasks.exe", "/Create", "/TN", updateTaskName, "/XML", xmlPath, "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("注册自动更新计划任务失败：%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (a *App) unregisterUpdateTask() error {
	registered, _ := a.updateTaskRegistered()
	if !registered {
		return nil
	}
	output, err := backgroundCommand("schtasks.exe", "/Delete", "/TN", updateTaskName, "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除自动更新计划任务失败：%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func (a *App) updateTaskRegistered() (bool, error) {
	err := backgroundCommand("schtasks.exe", "/Query", "/TN", updateTaskName).Run()
	return err == nil, nil
}

func scheduledRelaunchAllowed() bool {
	return true
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return replacer.Replace(value)
}
