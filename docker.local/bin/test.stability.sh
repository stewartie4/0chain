#!/bin/bash

bin_dir=$(dirname $0)

function start_miner() {
  cd $bin_dir/../miner$i && $bin_dir/start.b0miner.sh
}

function start_sharder() {
  cd $bin_dir/../sharder$i && $bin_dir/start.b0sharder.sh
}

function start_0chain() {
  for i in $(seq 1 $1)
  do
    start_sharder &
  done
  for i in $(seq 1 $2)
  do
    start_miner &
  done
}

function stop_miner() {
  MINER=$1 docker-compose -p miner$1 -f $bin_dir/../build.miner/docker-compose.yml stop
}

function stop_sharder() {
  SHARDER=$1 docker-compose -p sharder$1 -f $bin_dir/../build.sharder/docker-compose.yml stop
}

function test_case_1() {
  sudo $bin_dir/clean.sh
  start_0chain 1 3
  sleep 120
  stop_miner 1
  start_miner 4
  sleep 120
  $bin_dir/stop_all.miner.sh
  $bin_dir/stop_all.sharder.sh
}

cd $bin_dir/../../
test "$(basename $PWD)" = "0chain" || exit 1
test_case_1
cd -
