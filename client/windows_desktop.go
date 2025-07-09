//go:build windows

package main

import (
	"embed"

	"fyne.io/systray"
)

//go:embed resources
var resources embed.FS

func onReady() {
	icon, err := resources.ReadFile("resources/tray/tray_logo_inactive.ico")
	if err != nil {
		panic(err)
	}
	systray.SetIcon(icon)
	systray.SetTooltip("NursorGateway")
	systray.SetTitle("NursorGateway")

	mainPage := systray.AddMenuItem("主页面", "主页面")
	go func() {
		for range mainPage.ClickedCh {
			print("mainPage clicked")
		}
	}()

	systray.AddSeparator()

	start := systray.AddMenuItem("启动", "启动")
	go func() {
		for range start.ClickedCh {
			print("start clicked")
			activeIcon, err := resources.ReadFile("resources/tray/tray_logo.ico")
			if err != nil {
				panic(err)
			}
			systray.SetIcon(activeIcon)
		}
	}()

	stop := systray.AddMenuItem("停止", "停止")
	go func() {
		for range stop.ClickedCh {
			print("stop clicked")
			inactiveIcon, err := resources.ReadFile("resources/tray/tray_logo_inactive.ico")
			if err != nil {
				panic(err)
			}
			systray.SetIcon(inactiveIcon)
		}
	}()

	exit := systray.AddMenuItem("退出", "退出")
	exitIcon, err := resources.ReadFile("resources/tray/exit.png")
	if err != nil {
		panic(err)
	}
	exit.SetIcon(exitIcon)
	go func() {
		for range exit.ClickedCh {
			print("exit clicked")
		}
	}()

	systray.AddSeparator()

	trayMenu := systray.AddMenuItem("设置", "设置")
	authLaunch := trayMenu.AddSubMenuItem("开机自启动", "开机自启动")
	more := trayMenu.AddSubMenuItem("更多", "更多")
	go func() {
		for range authLaunch.ClickedCh {
			print("trayMenu clicked")
		}
	}()

	go func() {
		for range more.ClickedCh {
			print("more clicked")
		}
	}()

	println("onReady")
}

func onExit() {
	println("onExit")
}

func RunWindowsDesktop() {

	systray.Run(onReady, onExit)

}
