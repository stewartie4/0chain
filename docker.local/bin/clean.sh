#!/bin/sh

local_dir=$(dirname $0)/..
s=$1
m=$2
test -z $m && m=3
test -z $m && m=7

for i in $(seq 1 $m)
do
  echo "deleting miner$i logs"
  rm -rf $local_dir/miner$i/log/*
  echo "deleting miner$i redis db"
  rm -rf $local_dir/miner$i/data/redis/state/*
  rm -rf $local_dir/miner$i/data/redis/transactions/*
  echo "deleting miner$i rocksdb db"
  rm -rf $local_dir/miner$i/data/rocksdb/*
done

for i in $(seq 1 $s)
do
  echo "deleting sharder$i logs"
  rm -rf $local_dir/sharder$i/log/*
  echo "deleting sharder$i cassandra db"
  rm -rf $local_dir/sharder$i/data/cassandra/*
  echo "deleting sharder$i rocksdb db"
  rm -rf $local_dir/sharder$i/data/rocksdb/*
done

for i in $(seq 1 $s)
do
  echo "deleting sharder$i blocks on the file system"
  rm -rf $local_dir/sharder$i/data/blocks/*
done
