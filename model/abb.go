package model

import "fmt"

// function id description (yes, abb sends the id as string)
const (
	FID_SWITCH_SENSOR                                  = 0x0000
	FID_DIMMING_SENSOR                                 = 0x0001
	FID_BLIND_SENSOR                                   = 0x0003
	FID_SWITCH_ACTUATOR                                = 0x0007
	FID_SHUTTER_ACTUATOR                               = 0x0009
	FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITH_FAN    = 0x000A
	FID_ROOM_TEMPERATURE_CONTROLLER_SLAVE              = 0x000B
	FID_DIMMING_ACTUATOR                               = 0x0012
	FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN = 0x0023
	FID_BLIND_ACTUATOR                                 = 0x0061
	FID_ATTIC_WINDOW_ACTUATOR                          = 0x0062
	FID_AWNING_ACTUATOR                                = 0x0063
	FID_SMOKE_DETECTOR                                 = 0x007d
	FID_LightGroup                                     = 0x4000
	FID_DimmerGroup                                    = 0x4001
	FID_TimerProgramSwitchActuator                     = 0x4A00
	FID_Scence                                         = 0x4a01
	FID_DES_LEVEL_CALL_SENSOR                          = 0x001e
	FID_DES_LEVEL_CALL_ACTUATOR                        = 0x001D
	FID_DES_AUTOMATIC_DOOR_OPENER_ACTUATOR             = 0x0020
	FID_DES_DOOR_OPENER_ACTUATOR                       = 0x001A
	FID_DES_LIGHT_SWITCH_ACTUATOR                      = 0x0021
	FID_PANEL_SWITCH_SENSOR                            = 0x0030
	FID_PANEL_BLIND_SENSOR                             = 0x0033
	FID_PANEL_SCENE_SENSOR                             = 0x0037
	FID_DES_DOOR_RINGING_SENSOR                        = 0x001F
	FID_PANEL_ROOM_TEMPERATURE_CONTROLLER_SLAVE        = 0x0038
	FID_HEATING_ACTUATOR                               = 0x0027
	FID_RADIATOR_THERMOSTAT                            = 0x003f
	FID_COOLING_ACTUATOR                               = 0x0024
	FID_SCENE_SENSOR                                   = 0x0006
	FID_MOVEMENT_DETECTOR                              = 0x0011
	FID_WINDOW_DOOR_SENSOR                             = 0x000F
	FID_WINDOW_DOOR_POSITION_SENSOR                    = 0x0064
	FID_HUE_ACTUATOR                                   = 0x002E
	FID_SCENE                                          = 0x4800
	FID_SPECIAL_SCENE_PANIC                            = 0x4801
	FID_SPECIAL_SCENE_ALL_OFF                          = 0x4802
	FID_SPECIAL_SCENE_ALL_BLINDS_UP                    = 0x4803
	FID_SPECIAL_SCENE_ALL_BLINDS_DOWN                  = 0x4804
)

func GetFunctionIDsList() []string {
	constants := []int{
		FID_SWITCH_SENSOR,
		FID_DIMMING_SENSOR,
		FID_BLIND_SENSOR,
		FID_SWITCH_ACTUATOR,
		FID_SHUTTER_ACTUATOR,
		FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITH_FAN,
		FID_ROOM_TEMPERATURE_CONTROLLER_SLAVE,
		FID_DIMMING_ACTUATOR,
		FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN,
		FID_BLIND_ACTUATOR,
		FID_ATTIC_WINDOW_ACTUATOR,
		FID_AWNING_ACTUATOR,
		FID_SMOKE_DETECTOR,
		FID_LightGroup,
		FID_DimmerGroup,
		FID_TimerProgramSwitchActuator,
		FID_Scence,
		FID_DES_LEVEL_CALL_SENSOR,
		FID_DES_LEVEL_CALL_ACTUATOR,
		FID_DES_AUTOMATIC_DOOR_OPENER_ACTUATOR,
		FID_DES_DOOR_OPENER_ACTUATOR,
		FID_DES_LIGHT_SWITCH_ACTUATOR,
		FID_PANEL_SWITCH_SENSOR,
		FID_PANEL_BLIND_SENSOR,
		FID_PANEL_SCENE_SENSOR,
		FID_DES_DOOR_RINGING_SENSOR,
		FID_PANEL_ROOM_TEMPERATURE_CONTROLLER_SLAVE,
		FID_HEATING_ACTUATOR,
		FID_RADIATOR_THERMOSTAT,
		FID_COOLING_ACTUATOR,
		FID_SCENE_SENSOR,
		FID_MOVEMENT_DETECTOR,
		FID_WINDOW_DOOR_SENSOR,
		FID_WINDOW_DOOR_POSITION_SENSOR,
		FID_HUE_ACTUATOR,
		FID_SCENE,
		FID_SPECIAL_SCENE_PANIC,
		FID_SPECIAL_SCENE_ALL_OFF,
		FID_SPECIAL_SCENE_ALL_BLINDS_UP,
		FID_SPECIAL_SCENE_ALL_BLINDS_DOWN,
	}

	hexStrings := make([]string, len(constants))
	for i, constant := range constants {
		hexStrings[i] = fmt.Sprintf("%04X", constant)
	}
	return hexStrings
}

// *************************** VALUE MAP IN ELIONA
// 0 - up
// 1 - down
// 0x01 - comfort mode
// 0x02 - standby
// 0x04 - eco mode
// 0x08 - building protect
// 0x10 - dew alarm
// 0x20 - heat (set) / cool (unset)
// 0x40 - no heating/cooling (set)
// 0x80 - frost alarm

// 0 - off (Protection mode)
// 1 - on (Comfort mode or Eco mode)

// 0 - Off
// 1 - On

// 0 - not moving
// 2 - moves up
// 3 - moves down

const (
	PID_SWITCH_ON_OFF_SET                          = 0x0001
	PID_TIMED_START_STOP                           = 0x0002
	PID_AL_SCENE_CONTROL                           = 0x0004
	PID_PRESENCE                                   = 0x0007
	PID_AL_RELATIVE_SET_VALUE_CONTROL              = 0x0010
	PID_ABSOLUTE_VALUE_0_100_SET                   = 0x0011
	PID_BLINDER_UP_DOWN_SET                        = 0x0020
	PID_BLINDER_STOP_SET                           = 0x0021
	PID_BLINDER_ABS_POSITION_0_100_SET             = 0x0023
	PID_SET_ABSOLUTE_POSITION_SLATS_PERCENTAGE     = 0x0024
	PID_ACTUATING_VALUE_HEATING                    = 0x0030
	PID_AL_INFO_VALUE_HEATING                      = 0x0131
	PID_SETPOINT_TEMPERATURE_GET                   = 0x0033
	PID_HEATING_MODE_GET                           = 0x0036
	PID_CONTROLLER_ON_OFF_PROTECTED_GET            = 0x0038
	PID_CONTROLLER_ECOMODE_SET                     = 0x003A
	PID_CONTROLLER_REQ_ON_OFF_SET                  = 0x0042
	PID_ON_OFF_INFO_GET                            = 0x0100
	PID_ACTUAL_DIM_VALUE_0_100_GET                 = 0x0110
	PID_UP_DOWN_STOP_STATE                         = 0x0120
	PID_CURRENT_POSITION_BLIND_0_100_GET           = 0x0121
	PID_CURRENT_ABSOLUTE_POSITION_SLATS_PERCENTAGE = 0x0122
	PID_MEASURED_TEMPERATURE                       = 0x0130 // indicator ??
	PID_ABS_TEMPERATURE_SET                        = 0x0140 // set reg temp on room controller
	PID_HEATING_ACTIVE                             = 0x014B
	PID_HEATING_VALUE                              = 0x0131
	PID_AL_MOVEMENT_DETECTOR_STATUS                = 0x0166
	PID_MOVEMENT_UNDER_CONSIDERATION_OF_BRIGHTNESS = 0x0006
	PID_AL_WINDOW_DOOR                             = 0x0035
	PID_AL_WINDOW_DOOR_POSITION                    = 0x0029
	PID_HSV_COLOR_GET                              = 0x011B
	PID_HSV_HUE_SET                                = 0x0018
	PID_HSV_SATURATION_SET                         = 0x0019
	PID_HSV_VALUE_SET                              = 0x001A
	PID_COLOR_MODE_GET                             = 0x011D
	PID_COLOR_TEMPERATURE_GET                      = 0x0118
	PID_COLOR_TEMPERATURE_SET                      = 0x0016
)
