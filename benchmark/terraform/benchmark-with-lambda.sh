#!/bin/bash
for ((i=1;i<=10;i++)); 
do 
  aws lambda invoke --function-name benchmark  --region us-east-1 --invocation-type RequestResponse --payload '{ "type": "bitswap", "concurrency": 30, "iterations": 150 }' /dev/stdout &
done

wait