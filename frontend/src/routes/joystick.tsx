import { createRoute } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";
import { Ps4Panel } from "../components/panel/Ps4Panel";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/joystick",
  component: JoystickPage,
});

function JoystickPage() {
  // 使用 'global' 作为 gameId 来表示全局手柄设置
  return <Ps4Panel gameId="global" />;
}
