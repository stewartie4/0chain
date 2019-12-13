#!/bin/bash
#set -x
#MINER=3
#SHARDER=2
#BLOBBER=4
#PUBLIC_ENDPOINT=example.com
#MPORT=707
#SPORT=717
key_gen() {
echo "${5}s:" >> /config/nodes.yaml
for n in $(seq 1 $1); do
 echo -e "Creating keys for $5-${n}.. \n"
 go run key_gen.go  --signature_scheme "bls0chain" --keys_file_name "b0$4node${n}_keys.txt" --keys_file_path "/ms-keys" --generate_keys true  --timestamp true >> /config/nodes.yaml
status=$?
if [[ "$status" -eq "0" ]]; then
cat <<EOF >>/config/nodes.yaml
  n2n_ip: $5-${n}              
  public_ip: $2                   
  port: ${3}${n}             
  description: localhost.$4${n} 

EOF
else
 echo "Key generation failed"
 exit $retValue
fi

done
}
if [[ "$MINER" -ne "0" ]]; then
echo -e "Creating keys for miners \n"
key_gen $MINER $PUBLIC_ENDPOINT $MPORT m miner
fi
 
if [[ "$SHARDER" -ne "0" ]]; then
echo -e "Creating keys for sharders \n"
key_gen $SHARDER $PUBLIC_ENDPOINT $SPORT s sharder
fi
if [[ "$BLOBBER" -ne "0" ]]; then
 echo -e "Creating keys for Blobbers.. \n"
 for n in $(seq 1 $BLOBBER); do
  echo -e "Creating keys for blobber-${n}.. \n"
  go run key_gen.go --signature_scheme "bls0chain" --keys_file_name "b0bnode${n}_keys.txt" --keys_file_path "/blob-keys" --generate_keys true  --timestamp true > /dev/null 2>&1 
 done
fi
if [[ ! -z "$ZBOX" ]]; then
 echo -e "Creating keys for 0box.. \n"
go run key_gen.go --signature_scheme "bls0chain" --keys_file_name "0box_keys_bls.txt" --keys_file_path "/zbox-keys" --generate_keys true  --timestamp true > /dev/null 2>&1
cat <<EOF >>/zbox-keys/0box_keys_bls.txt
$PUBLIC_ENDPOINT
$ZPORT
EOF
fi
#exec $@
