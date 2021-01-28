#!/bin/bash
input="nginx_json_logs"
while IFS= read -r line 
do
    echo "$line"
    echo "$line" >> "log_sample/app_nginx_logs"
done < "$input"
