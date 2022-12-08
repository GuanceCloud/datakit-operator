#!/bin/bash

host="$1"
bucket="$2"
id="$3"
key="$4"
version="$5"
osshost=$bucket.$host
echo $osshost

# check input parameters
if [ -z "$host" ]; then
	echo "invalid host parameter, exit"
	exit 1
elif [ -z "$bucket" ]; then
         echo "invalid bucket parameter, exit"
         exit 1
elif [ -z "$id" ]; then
         echo "invalid id parameter, exit"
         exit 1
elif [ -z "$key" ]; then
         echo "invalid key parameter, exit"
         exit 1
elif [ -z "$version" ]; then
	echo "invalid version parameter, exit"
	exit 1
fi

name="datakit-operator"
source="datakit-operator.yaml"
sourceWithVersion="${name}-${version}.yaml"

dest="${name}/${source}"
destV="${name}/${sourceWithVersion}"

dateValue="`TZ=GMT env LANG=en_US.UTF-8 date +'%a, %d %b %Y %H:%M:%S GMT'`"


# 1 upload no version file
resource="/${bucket}/${dest}"
contentType=`file -ib ${source} | awk -F ";" '{print $3}'`
stringToSign="PUT\n\n${contentType}\n${dateValue}\n${resource}"
signature=`echo -en $stringToSign | openssl sha1 -hmac ${key} -binary | base64`

url=http://${osshost}/${dest}
echo "upload ${source} to ${url}"

curl -i -q -X PUT -T "${source}" \
    -H "Host: ${osshost}" \
    -H "Date: ${dateValue}" \
    -H "Content-Type: ${contentType}" \
    -H "Authorization: OSS ${id}:${signature}" \
    ${url}


# 2 upload has version file
resource="/${bucket}/${destV}"
contentType=`file -ib ${sourceWithVersion} | awk -F ";" '{print $3}'`
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

