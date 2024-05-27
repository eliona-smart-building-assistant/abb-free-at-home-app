# ABB Free@home

### Eliona App for ABB Free@home integration

> Simply smart. ABB-free@home® transforms the house or the apartment into an intelligent home. Whether blinds, lights, heating, air conditioning, door communication or scenes. Easy to remote control via a switch on the wall, with the laptop or with the smartphone. Very convenient. Extremely comfortable. Very energy efficient. Especially attractive: Only minimal costs are involved when compared with conventional electrical installations.

This app allows accessing ABB Free@home systems directly in Eliona using the ABB ProService portal. Users can monitor values, browse through statistics, control the Free@home devices and, most importantly, interconnect ABB devices with systems from other manufacturers.

## Installation

The ABB Free@home App is installed via the App Store in Eliona.

## Assets

The ABB Free@home App automatically creates all the necessary asset types and assets. The asset names are kept synchronized from ABB -> Eliona (therefore assets must be renamed in Free@home system. If they are renamed in Eliona, the name will not ve persisted).

### Structure assets
The following asset types are created just to create a structure in Eliona:

- *Floor*: Represents a specific level in a building.

| Attribute | Description       |
|-----------|-------------------|
| `Id`      | Floor Identifier  |
| `Name`    | Floor Name        |
| `Level`   | Floor Level       |

- *Room*: Represents a specific room on a floor.

| Attribute | Description     |
|-----------|-----------------|
| `Id`      | Room Identifier |
| `Name`    | Room Name       |

- *System*: Represents a central system controlling multiple devices.

| Attribute | Description  | Filterable |
|-----------|--------------|------------|
| `ID`      | System ID    | x          |
| `GAI`     | GAI          | x          |
| `Name`    | System Name  | x          |

- *Device*: Represents a specific device in the system. Devices are linked to their respective systems and locations in Eliona asset tree.

| Attribute | Description        | Filterable |
|-----------|--------------------|------------|
| `ID`      | Device Identifier | x          |
| `GAI`     | GAI               |            |
| `Name`    | Device Name       | x          |
| `Location`| Device Location   |            |
| `Battery` | Battery percentage (if applicable) |  |
| `Connectivity`| Connectivity status (if applicable) |  |

### Channels
Channels are linked to devices. These channels provide the real functionality:

Here's the updated documentation based on the provided data structure:

- *Switch*: A regular light switch.

| Attribute | Description | Subtype |
|-----------|-------------|---------|
| `Switch`  | Switch      | output  |

- *Dimmer*: A channel to control lighting intensity.

| Attribute | Description | Subtype |
|-----------|-------------|---------|
| `Switch`  | Switch      | output  |
| `Dimmer`  | Dimmer      | output  |

- *HueActuator*: A channel to control colored lighting.

| Attribute            | Description           | Subtype |
|----------------------|-----------------------|---------|
| `HSVState`           | HSV State             | input   |
| `ColorModeState`     | Color Mode State      | input   |
| `Switch`             | Switch                | output  |
| `Dimmer`             | Dimmer                | output  |
| `HSVHue`             | HSV Hue               | output  |
| `HSVSaturation`      | HSV Saturation        | output  |
| `HSVValue`           | HSV Value             | output  |
| `ColorTemperature`   | Color Temperature     | output  |

- *RTC*: Room Temperature Controller.

| Attribute     | Description          | Subtype |
|---------------|----------------------|---------|
| `CurrentTemp` | Current Temperature  | input   |
| `Switch`      | Switch               | output  |
| `SetTemp`     | Set Temperature      | output  |

- *RadiatorThermostat*: Thermostat for a radiator.

| Attribute          | Description          | Subtype |
|--------------------|----------------------|---------|
| `CurrentTemp`      | Current Temperature  | input   |
| `StatusIndication` | Status Indication    | input   |
| `HeatingActive`    | Heating Active       | input   |
| `HeatingValue`     | Heating Value        | input   |
| `Switch`           | Switch               | output  |
| `SetTemp`          | Set Temperature      | output  |

- *HeatingActuator*: Heating control unit.

| Attribute      | Description   | Subtype |
|----------------|---------------|---------|
| `InfoFlow`     | Info Flow     | input   |
| `ActuatorFlow` | Actuator Flow | input   |

- *WindowSensor*: Sensor to detect window position.

| Attribute  | Description | Subtype |
|------------|-------------|---------|
| `Position` | Position    | input   |

- *DoorSensor*: Sensor to detect door position.

| Attribute  | Description | Subtype |
|------------|-------------|---------|
| `Position` | Position    | input   |

- *MovementSensor*: Sensor to detect movement.

| Attribute  | Description | Subtype |
|------------|-------------|---------|
| `Movement` | Movement    | input   |

- *SmokeDetector*: Sensor to detect fire.
Note: ABB implements it in a way that makes all fire detectors trigger an alarm at once when one device detects smoke.

| Attribute | Description | Subtype |
|-----------|-------------|---------|
| `Fire`    | Fire        | input   |

- *FloorCallButton*: Rings a bell.

| Attribute  | Description | Subtype |
|------------|-------------|---------|
| `FloorCall`| Floor Call  | output  |

- *MuteButton*: Mutes the floor call button.

| Attribute | Description  | Subtype |
|-----------|--------------|---------|
| `Mute`    | Mute Button  | output  |

- *Scene*: Represents a scene.

| Attribute | Description  | Subtype |
|-----------|--------------|---------|
| `Switch`  | Set Scene    | output  |

- *Wallbox*: Car charging station.

| Attribute           | Description          | Subtype |
|---------------------|----------------------|---------|
| `Switch`            | Switch               | output  |
| `Enable`            | Enable               | output  |
| `InstalledPower`    | Installed Power      | status  |
| `TotalEnergy`       | Total Energy         | input   |
| `StartLastCharging` | Start Last Charging  | input   |
| `Status`            | Status               | input   |

## Configuration

The ABB Free@home App is configured by defining one or more authentication credentials. Each configuration requires the following data:

| Attribute        | Description                                               |
|------------------|-----------------------------------------------------------|
| `abbConnectionType`  | Type of connection. Only "ProService" is currently supported. |
| `apiKey`       | API key provided by ABB                        |
| `orgUUID`   | UUID of the ProService organization                   |
| `enable`         | Flag to enable or disable fetching from this API          |
| `refreshInterval`| Interval in seconds for device discovery. This is an expensive operation, should be no lower than 3600 s |
| `requestTimeout` | API query timeout in seconds                              |
| `assetFilter`    | Filter for asset creation, more details can be found in app's README |
| `projectIDs`     | List of Eliona project ids for which this device should collect data. For each project id, all assets are automatically created in Eliona. |

The configuration is done via a corresponding JSON structure. As an example, the following JSON structure can be used to define an endpoint for app permissions:

```
{
  "abbConnectionType": "ProService",
  "apiKey": "api.key",
  "orgUUID": "org-uuid",
  "enable": true,
  "refreshInterval": 3600,
  "requestTimeout": 120,
  "assetFilter": [],
  "projectIDs": [
    "10"
  ]
}
```

Configurations can be created using this structure in Eliona under `Apps > ABB Free@home > Settings`. To do this, select the /configs endpoint with the POST method.

After completing configuration, the app starts Continuous Asset Creation. When all discovered devices are created, user is notified about that in Eliona's notification system.

## After configuration

After the application is configured, it looks up systems connected to the configured ProService account. On all of these systems, it automatically creates a user called "eliona_ProService" that would later be used when controlling the devices. This account has to be enabled locally on these systems.

To enable the account, log in to the SysAPs, find "User settings" and find a user called "eliona_ProService". Enable this user and ensure it has the correct access rights to control the devices.

## Troubleshooting

### Defective Device error message

There is an error with some devices, that some datapoint writes cause the system to consider the device "defective". The system then stops sending data to these devices, making them uncontrollable.

The devices are not defective though, the sysAP just needs to be restarted and the devices respond again.

ABB is aware of that error and on a way to fix it. If you run into this issue repeatedly, please let us know about it for a fix.

### Troubleshooting using GraphQL

> ABB has a GraphQL playground for it's Smart Home API: https://apim.eu.mybuildings.abb.com/adtg-api/v1/graphiql

You can log in to the playground either using your MyBuildings account or your ProService account.

```
{
  PSOrganization{dtId} # should not work for a normal user
}
{
  User{userName} # should return the username of the user who created the token
}
```

To verify which relation you have to the systems, you can use this query:
```
{
  UserDevice  {dtId} #the dtId should appear here - in case the user is the owner of the sysap
  CustomerDevice {dtId} #the dtId should appear here - in case the user is the installer that invited the user as customer
}
```

Please note that ABB currently does not recommend using the same user account for ProService and MyBuildings portals. It causes some problems that can be worked around, but are not desired.

### Requests optimization

ABB implemented a way to analyze resource usage by current user:
```
{
  ServerDescriptionService
  {
    consumedRequestCosts # returns the sum of costs produced by this user (should be turned back to 0 in the night)
    requestCosts #returns the costs by this query (in your case mainly affected by 1 per object loaded + 1 per DataPointRequest)
  }
# … add here the rest/normal query
}
```

We took a lot of effort to bring resource usage down. Still, the app polling for new devices is a very resource-intensive process that can be further optimized if there is a demand for it.

