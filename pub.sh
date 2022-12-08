#!/bin/bash

host="oss-cn-hangzhou.aliyuncs.com"
bucket="df-storage-dev" # BucketName
id="$1" # AccessKeyId
key="$2" # AccessKeySecret
osshost=$bucket.$host
version="$3"
echo $osshost

name="datakit-operator"
source="datakit-operator.yaml"
sourceWithVersion="${name}-${version}.yaml"

dest="${name}/${source}"
destV="${name}/${sourceWithVersion}"

dateValue="`TZ=GMT env LANG=en_US.UTF-8 date +'%a, %d %b %Y %H:%M:%S GMT'`"

# 1 upload no version jar file.
resource="/${bucket}/${dest}"
contentType=`file -ib ${agentName} | awk -F ";" '{print $1}'`
stringToSign="PUT\n\n${contentType}\n${dateValue}\n${resource}"
signature=`echo -en $stringToSign | openssl sha1 -hmac ${key} -binary | base64`


url=http://${osshost}/${dest}
echo "upload ${agentName} to ${url}"


curl -i -q -X PUT -T "${agentName}" \
    -H "Host: ${osshost}" \
    -H "Date: ${dateValue}" \
    -H "Content-Type: ${contentType}" \
    -H "Authorization: OSS ${id}:${signature}" \
    ${url}

# 2 upload has version jar file.
resource="/${bucket}/${destV}"
contentType=`file -ib ${sourceWithVersion} | awk -F ";" '{print $1}'`
stringToSign="PUT\n\n${contentType}\n${dateValue}\n${resource}"
signature=`echo -en $stringToSign | openssl sha1 -hmac ${key} -binary | base64`


url=http://${osshost}/${destV}
echo "upload ${sourceWithVersion} to ${url}"


curl -i -q -X PUT -T "${sourceWithVersion}" \
    -H "Host: ${osshost}" \
    -H "Date: ${dateValue}" \
    -H "Content-Type: ${contentType}" \
    -H "Authorization: OSS ${id}:${signature}" \
    ${url}

