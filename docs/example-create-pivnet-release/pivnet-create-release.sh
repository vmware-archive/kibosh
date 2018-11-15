#!/bin/bash

REFRESH_TOKEN="aaaabbbccccddddeeeeffffffffff-a" # Get your refresh token from your Pivotal Network profile
PRODUCT_SLUG_NAME="product-slug"
NEW_VERSION="0.0.0.a"
ECCN="5D002" # Export Control Classification Number
RELEASE_DESCRIPTION="A short description of this release."
RELEASE_TYPE="Beta Release"
RELEASE_DATE="2018-09-30"
END_OF_SUPPORT="2018-12-30"
END_OF_GUIDANCE="2018-12-30"
END_OF_AVAILABILITY="2018-12-30"
SRC_DIR="./release-files"



## Don't edit below this line

# Get a refresh token
PIVNET_ACCESS_TOKEN=`curl -s https://network.pivotal.io/api/v2/authentication/access_tokens -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq -r '.access_token'`

if [ -z "$PIVNET_ACCESS_TOKEN" ] || [ "$PIVNET_ACCESS_TOKEN" = "null" ]
    then
        echo "Error getting access token."
        exit 1
fi

# Get Product ID
PRODUCT_ID=`curl -s -X GET https://network.pivotal.io/api/v2/products -H "Authorization: Bearer $PIVNET_ACCESS_TOKEN" | jq --arg PRODUCT_SLUG_NAME "$PRODUCT_SLUG_NAME" -c '.products[] | select(.slug ==$PRODUCT_SLUG_NAME) | .id'`
echo "Product ID: $PRODUCT_ID"

if [ -z "$PRODUCT_ID" ]
    then
        echo "Error finding product [$PRODUCT_SLUG_NAME]."
        exit 1
fi

# Create a release
RELEASE_ID=`curl https://network.pivotal.io/api/v2/products/$PRODUCT_ID/releases -H "Authorization: Bearer $PIVNET_ACCESS_TOKEN" -d "{\"copy_metadata\":true,\"release\":{\"version\":\"$NEW_VERSION\",\"release_notes_url\":\"http://example.com/release\",\"description\":\"$RELEASE_DESCRIPTION\",\"release_date\":\"$RELEASE_DATE\",\"release_type\":\"$RELEASE_TYPE\",\"availability\":\"Admins Only\",\"eula\":{\"slug\":\"pivotal_software_eula\"},\"oss_compliant\":\"confirm\",\"end_of_support_date\":\"$END_OF_SUPPORT\",\"end_of_guidance_date\":\"$END_OF_GUIDANCE\",\"end_of_availability_date\":\"$END_OF_AVAILABILITY\",\"eccn\":\"$ECCN\",\"license_exception\":\"ENC Unrestricted\"}}" | jq -r '.release.id'`
echo "Release Id: $RELEASE_ID"

if [ -z "$RELEASE_ID" ] || [ "$RELEASE_ID" = "null" ] ; then
        echo "Error create release. Release [$NEW_VERSION] may already exist."
        exit 1
fi

# Get federation token
IFS=',' read AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN s3Bucket s3Region < <(curl https://network.pivotal.io/api/v2/federation_token -d "{\"product_id\": \"$PRODUCT_SLUG_NAME\"}"  -H "Authorization: Bearer $PIVNET_ACCESS_TOKEN" | jq -r '[.access_key_id, .secret_access_key, .session_token, .bucket, .region] | @csv' | sed 's/"//g')
export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY
export AWS_SESSION_TOKEN


# Get S3 path
relativePath=`curl -s -X GET https://network.pivotal.io/api/v2/products/$PRODUCT_SLUG_NAME -H "Authorization: Bearer $PIVNET_ACCESS_TOKEN" | jq -r '.s3_directory.path'`
relativePath="${relativePath:1}"
cd "$SRC_DIR"
for filename in *; do
    [ -e "$filename" ] || continue

    # Upload Product File(s) to S3
    echo "uploading $filename to s3"
    aws s3 cp $filename s3://$s3Bucket/$relativePath/$filename --region $s3Region

    # Get sha256 hash
    FILE_SHA256=`openssl sha256 -hex $filename | cut -f2 -d ' '`

    # Add product files/ Get Product File ID(s)
    PRODUCT_FILE_ID=`curl -X POST https://network.pivotal.io/api/v2/products/$PRODUCT_ID/product_files -H "Authorization: Bearer ${PIVNET_ACCESS_TOKEN}" -d "{\"product_file\":{\"aws_object_key\":\"$relativePath/$filename\",\"description\":\"$filename\",\"docs_url\":\"http://example.com\",\"file_type\":\"Software\",\"file_version\":\"0.0.1\",\"included_files\":[\"$filename\"],\"sha256\":\"$FILE_SHA256\",\"name\":\"$filename\",\"released_at\":\"2018/12/31\",\"system_requirements\":[\"Windows Vista\",\"Microsoft Office 1995\"]}}" | jq .product_file.id`
    if [ -z "$PRODUCT_FILE_ID" ]
    then
	echo "Error adding product file"
        exit 1
    else
        echo "Added file $filename. Product file ID: $PRODUCT_FILE_ID"
    fi

    # Add Product Files to release
    curl -X PATCH https://network.pivotal.io/api/v2/products/$PRODUCT_ID/releases/$RELEASE_ID/add_product_file -H "Authorization: Bearer ${PIVNET_ACCESS_TOKEN}" -d "{\"product_file\":{\"id\":$PRODUCT_FILE_ID}}"
done


