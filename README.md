![Linux](https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black) ![Windows](https://img.shields.io/badge/Windows-0078D6?style=for-the-badge)   ![MQTT](https://img.shields.io/badge/-MQTT-%238D6748?style=for-the-badge)
 
# gohostmon
This is a simple Go utility built on top of the paho.mqtt.golang and gopsutil libraries to monitor the host machine running Home Assistant. The monitored machine is a Dell Wyse 5060 with an AMD K10 CPU, but the script can be easily modified to work with other devices as well.

The program uses MQTT protocol and will publish everything under hwinfo/ topic, as follows

```
hwinfo/k10_temperature_celsius
hwinfo/cpu_utilization_percent
hwinfo/ram_used_percent
```

Then, you should create configuration file named config.ini in the working directory where you run the script:
```ini
[credentials]
user=<your mqtt username>
pass=<your mqtt password>
host=<your mqtt hostname>
```
Then, you should create a bash script, which you'll put somewhere to automatically start this python script on boot.

# Running
This is an example autorun script dedicated to Kubuntu autorun mechanism which also launches HA instance:
```bash
#!/bin/bash

cd /home/chris/Pulpit/HomeAssistant/uhostmon
nohup ./gohostmon >/dev/null 2>&1 &
VBoxManage startvm HomeAssistant --type headless
```

# Screenshots
- Home Assistant dasboard

![image](https://github.com/user-attachments/assets/d41f1c33-7706-48bf-9739-6853c9bd076b)

- MQTT explorer (see hwinfo topic)

![image](https://github.com/cziter15/uhostmon/assets/5003708/a11ea6b3-8fef-4883-8bea-9d8801351935)
