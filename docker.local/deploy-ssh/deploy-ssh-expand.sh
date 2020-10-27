#/bin/sh

# run inside remote 0chain directory

set -x

##
## 1st and the singe argument must be external IP address of the server
##

# patch localhost -> real IP address
ip_address="${@}"

if [ -z "${ip_address}" ]; then
	echo "use: sh docker.local/bin/deploy-ssh-expand.sh 'ip_address'"
	exit 1
fi

echo "IP address: ${ip_address}"

###
### patch localhost to given IP (script argument)
###

echo "patch localhost to given IP address"

# patch magic block file
temp_mb=$(mktemp)
jq '.miners.nodes[].host = "'"${ip_address}"'"' docker.local/config/b0magicBlock_4_miners_1_sharder.json > "${temp_mb}"
mv -v "${temp_mb}" docker.local/config/b0magicBlock_4_miners_1_sharder.json

# patch *_keys.txt files for non-genesis nodes
for n in $(seq 2 3)
do
	sed -i 's/localhost/'"${ip_address}"'/g' "docker.local/config/b0snode${n}_keys.txt"
done
for n in $(seq 5 8)
do
	sed -i 's/localhost/'"${ip_address}"'/g' "docker.local/config/b0mnode${n}_keys.txt"
done

# setup nodes

echo "setup 0chain directories"
./docker.local/bin/init.setup.sh

echo "setup docker network"
./docker.local/bin/setup_network.sh

echo "stop current running, if any"
for i in $(seq 1 8)
do
  sudo systemctl stop "miner${i}" || true
done

for i in $(seq 1 3)
do
  sudo systemctl stop "sharder${i}" || true
done

echo "cleanup containers and volumes"
./docker.local/bin/docker-clean.sh

# systemd services
#

echo "create or update units"

# miners services
#

for i in $(seq 1 8)
do
  cat > miner${i}.service << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service
Description=0chain/miner${i}

[Service]
Type=simple
WorkingDirectory=$(pwd)/docker.local/miner${i}
User=$(id -nu)
Group=$(id -ng)
ExecStart=$(pwd)/docker.local/bin/start.b0miner.sh
ExecStop=$(pwd)/docker.local/bin/stop.b0miner.sh
TimeoutSec=30
RestartSec=15
Restart=always

[Install]
WantedBy=multi-user.target
EOF
	sudo mv -v miner${i}.service /etc/systemd/system/
done

# sharders services
#

for i in $(seq 1 3)
do
  cat > sharder${i}.service << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service
Description=0chain/sharder${i}

[Service]
Type=simple
WorkingDirectory=$(pwd)/docker.local/sharder${i}
User=$(id -nu)
Group=$(id -ng)
ExecStart=$(pwd)/docker.local/bin/start.b0sharder.sh
ExecStop=$(pwd)/docker.local/bin/stop.b0sharder.sh
TimeoutSec=180
RestartSec=15
Restart=always

[Install]
WantedBy=multi-user.target
EOF
	sudo mv -v sharder${i}.service /etc/systemd/system/
done

echo "reload systemd daemon"
sudo systemctl daemon-reload

echo "done, no units started, start them manually"
