package main

import (
	"fmt"
	"time"

	"github.com/0xcafed00d/joystick"
)

func main() {
	fmt.Println("等待检测手柄...")

	// 等待手柄连接
	var joy joystick.Joystick
	var err error

	for i := 0; i < 10; i++ {
		joy, err = joystick.Open(i)
		if err == nil {
			fmt.Printf("找到手柄: %s (Index: %d, Axes: %d, Buttons: %d)\n",
				joy.Name(), i, joy.AxisCount(), joy.ButtonCount())
			fmt.Println("\n请按下按钮或移动摇杆，观察下方输出：")
			fmt.Println("格式: Type:ID Value:Value")
			fmt.Println("按 Ctrl+C 退出\n")
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		fmt.Println("未找到手柄")
		return
	}

	// 持续读取状态
	prevButtons := make([]bool, joy.ButtonCount())
	prevAxes := make([]int, joy.AxisCount())

	for {
		state, err := joy.Read()
		if err != nil {
			fmt.Println("读取错误:", err)
			break
		}

		// 检测按钮变化
		for i := 0; i < joy.ButtonCount(); i++ {
			pressed := state.Buttons&(1<<i) != 0
			if pressed != prevButtons[i] {
				if pressed {
					fmt.Printf("Button:%d Pressed\n", i)
				} else {
					fmt.Printf("Button:%d Released\n", i)
				}
				prevButtons[i] = pressed
			}
		}

		// 检测轴变化
		for i := 0; i < joy.AxisCount(); i++ {
			if state.AxisData[i] != prevAxes[i] {
				fmt.Printf("Axis:%d Value:%d\n", i, state.AxisData[i])
				prevAxes[i] = state.AxisData[i]
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}
