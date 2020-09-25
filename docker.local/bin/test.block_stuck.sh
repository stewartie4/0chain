#!/bin/bash

s=1
m=4
result=0
bin_dir=$(dirname $(realpath $0))

function start_miner() {
  cd $bin_dir/../miner$1 && $bin_dir/start.b0miner.sh > /dev/null 2>&1
}

function start_sharder() {
  cd $bin_dir/../sharder$1 && $bin_dir/start.b0sharder.sh > /dev/null 2>&1
}

function stop_miner() {
  MINER=$1 docker-compose -p miner$1 -f $bin_dir/../build.miner/docker-compose.yml stop > /dev/null 2>&1
}

function stop_sharder() {
  SHARDER=$1 docker-compose -p sharder$1 -f $bin_dir/../build.sharder/docker-compose.yml stop > /dev/null 2>&1
}

function get_round() {
  curl -s http://localhost:7$1/_diagnostics | grep -o -P '.{0,1}Current Round.{0,30}' | grep -o -P '.{0,0}number.{0,30}' | cut -f 2 -d '>' | cut -f 1 -d '<'
}

function check_round() {
  local s1r=$(get_round 171)
  local m1r=$(get_round 071)
  local m2r=$(get_round 072)
  local m3r=$(get_round 073)
  local m4r=$(get_round 074)

  echo "node3 rounds: $m1r, $m2r, $m3r, $m4r, $s1r"

  test $s1r -lt $1 && return 300
  test $s1r -gt $2 && return 301
  if [ -n "$m1r" ]
  then
    test $m1r -lt $1 && return 302
    test $m1r -gt $2 && return 303
  fi
  test $m2r -lt $1 && return 304
  test $m2r -gt $2 && return 305
  test $m3r -lt $1 && return 306
  test $m3r -gt $2 && return 307
  if [ -n "$m4r" ]
  then
    test $m4r -lt $1 && return 308
    test $m4r -gt $2 && return 309
  fi 
}

function start_0chain() {
  for i in $(seq 1 $1)
  do
    start_sharder $i &
  done
  for i in $(seq 1 $2)
  do
    start_miner $i &
  done
}

function test_case_1() {
  start_0chain $s $(($m - 1))
  sleep 75
  check_round 65 115 || return $?
  echo "stopping m1 and s1 at $(docker exec miner3_miner_1 date)"
  stop_miner 1
  stop_sharder 1
  sleep 25
  echo "starting s1 and m4 at $(docker exec miner3_miner_1 date)"
  start_sharder 1 &
  start_miner $m &
  sleep 150
  check_round 130 230 || return $?
}

cd $bin_dir/../../
test "$(basename $PWD)" = "0chain" || exit 1
sudo $bin_dir/clean.sh $s $m
test_case_1 || result=$?
$bin_dir/stop_all.sharder.sh $s
$bin_dir/stop_all.miner.sh $m
test $result -eq 0 && echo "tests passed succesfully"
cd -
exit $result
