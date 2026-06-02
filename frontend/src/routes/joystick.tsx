import { createRoute } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";
import { KeyMappingPanel } from "../components/panel/KeyMappingPanel";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/joystick",
  component: JoystickPage,
});

function JoystickPage() {
  // 使用 'global' 作为 gameId 来表示全局手柄设置
  return <KeyMappingPanel gameId="global" />;
}
