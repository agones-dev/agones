#!/bin/bash

# Copyright Contributors to Agones a Series of LF Projects, LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

cd /go/src/github.com/ahmetb/gen-crd-api-reference-docs

#Use local version of agones
go mod edit --replace=agones.dev/agones@latest=../../../agones.dev/agones/
go mod tidy
go build 

cp /go/src/agones.dev/agones/site/assets/templates/pkg.tpl ./template

FILE=${FILE:-/go/src/agones.dev/agones/site/content/en/docs/Reference/agones_crd_api_reference.html}
VERSION=${VERSION:-"0.9.0"}

# +++ Title section
TITLE="/tmp/title.html"
# Current ExpiryVersion
EXPIRY_DOC="/tmp/expiry.html"
# Previous Publish Version after release
PUBLISH_DOC="/tmp/publish.html"
# Output of the gen-crd-api-reference-docs
RESULT="/tmp/agones_crd_api_reference.html"
# Version to compare
OLD="/tmp/old_docs.html"

cd /go/src/agones.dev/agones
../../github.com/ahmetb/gen-crd-api-reference-docs/gen-crd-api-reference-docs \
  -config ./site/assets/templates/crd-doc-config.json \
  -template-dir ../../github.com/ahmetb/gen-crd-api-reference-docs/template \
  -api-dir ./pkg/apis/ \
  -out-file $RESULT
awk '/\ feature\ publishVersion/{flag=1;next}/\ \/feature/{flag=0}flag' $FILE > $OLD

# Get the title lines from +++ till empty string
awk '/\+\+\+/ {ok=1} /^$/ {ok=0} {if(ok){print $0}}' $FILE > $TITLE
printf "\n" >> $TITLE

doc_version=$(grep 'feature publishVersion=' $FILE )
echo $doc_version
publish='{{% feature publishVersion="'$VERSION'" %}}'
expiry='{{% feature expiryVersion="'$VERSION'" %}}'

#Get previous expiry version tag
sed '/\ expiryVersion="'$VERSION'"/,/%\ \/feature/!d;/%\ \/feature/q' $FILE > $EXPIRY_DOC
sed '/\ publishVersion=/,/%\ \/feature/!d;/%\ \/feature/q' $FILE > $PUBLISH_DOC

function sedeasy {
  sed -i "s/$(echo $1 | sed -e 's/\([[\/.*]\|\]\)/\\&/g')/$(echo $2 | sed -e 's/[\/&]/\\&/g')/g" $3
}

# do we have changes in generated API docs compared to previous version
if ! diff <(sort $RESULT) <(sort $OLD);
then 
  echo "Output to a file $FILE"

  if [ "$publish" != "$doc_version" ]
  then 
    echo "Tagging old publishVersion section with expiryVersion shortcode"

    sedeasy "$doc_version" "$expiry"  $PUBLISH_DOC
    cat $PUBLISH_DOC >> $TITLE
  else 
    echo "expiry version left unchanged"
    cat $EXPIRY_DOC >> $TITLE
  fi
  cat $TITLE > $FILE
  echo -e '{{% feature publishVersion="'$VERSION'" %}}' >> $FILE
  cat $RESULT >> $FILE
  echo -e '{{% /feature %}}\n' >> $FILE
fi
