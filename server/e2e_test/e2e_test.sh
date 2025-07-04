#! /bin/bash

test_dir="./e2e_test_working_dir"
mkdir -p $test_dir
cd $test_dir

go build ../test_client/client.go 
go build ../test_server/server.go 

./server ../test_server/test_config.yml &
sleep 1
./client ./priv 20000
ret=$?
if [ $ret != 0 ]; then
    echo test failed
fi

kill $(jobs -p)

cd ..
rm -rf $test_dir

echo all test success
exit $ret