#!/bin/bash

function usage() {
    echo "Usage: $0 [flags] [flag-fields]"
    echo ""
    echo "Flags:"
    echo "    -d,   --deploy        prepare tar archives in build/, only works on a tagged commit in the format v1.0.0 or v1.0.0-rc1"
    echo "    -f,   --force         force deploy, combined with the -d flag to release for non-tagged commits"
    echo "    -t,   --test-deploy   test prepare tar archives in build/ for your native platform only"
    echo "    -h,   --help          show this help info"
}

# takes in the exit code ($?) of the previous command as the input
function check_build_result() {
    if [[ $1 -ne 0 ]]
    then
        echo ""
        echo "build failed with error code $1"
        exit $1
    fi
}

if [[ $(basename $("pwd")) != "kelp" && $("pwd") != *"circleci/project"* ]];
then
    echo "need to invoke from the root 'kelp' directory"
    exit 1
fi

KELP=`pwd`

if [[ ($# -eq 1 && ("$1" == "-d" || "$1" == "--deploy")) ]]; then
    ENV=release
    IS_TEST_MODE=0
    FORCE_RELEASE=0
elif [[ ($# -eq 1 && ("$1" == "-df" || "$1" == "-fd")) || ($# -eq 2 && ("$1" == "-d" || "$1" == "--deploy") && ("$2" == "-f" || "$2" == "--force")) || ($# -eq 2 && ("$1" == "-f" || "$1" == "--force") && ("$2" == "-d" || "$2" == "--deploy")) ]]; then
    ENV=release
    IS_TEST_MODE=0
    FORCE_RELEASE=1
elif [[ ($# -eq 1 && ("$1" == "-t" || "$1" == "--test-deploy")) ]]; then
    ENV=release
    IS_TEST_MODE=1
elif [[ ($# -eq 1 && ("$1" == "-h" || "$1" == "--help")) ]]; then
    usage
    exit 0
elif [[ $# -eq 0 ]]; then
    ENV=dev
else
    usage
    exit 1
fi

# version is git tag if it's available, otherwise git hash
VERSION=$(git describe --always --abbrev=8 --dirty --tags)
GIT_BRANCH=$(git branch | grep \* | cut -d' ' -f2)
VERSION_STRING="$GIT_BRANCH:$VERSION"
GIT_HASH=$(git describe --always --abbrev=50 --dirty --long)
DATE=$(date -u +%"Y%m%dT%H%M%SZ")
LDFLAGS_ARRAY=("github.com/stellar/kelp/cmd.version=$VERSION_STRING" "github.com/stellar/kelp/cmd.gitBranch=$GIT_BRANCH" "github.com/stellar/kelp/cmd.gitHash=$GIT_HASH" "github.com/stellar/kelp/cmd.buildDate=$DATE" "github.com/stellar/kelp/cmd.env=$ENV" "github.com/stellar/kelp/cmd.amplitudeAPIKey=$AMPLITUDE_API_KEY")

LDFLAGS=""
for FLAG in "${LDFLAGS_ARRAY[@]}"
do
    LDFLAGS="$LDFLAGS -X $FLAG"
done

echo "version: $VERSION_STRING"
echo "git branch: $GIT_BRANCH"
echo "git hash: $GIT_HASH"
echo "build date: $DATE"
echo "env: $ENV"
echo "LDFLAGS: $LDFLAGS"

if [[ $ENV == "release" ]]
then
    if [ -z "$AMPLITUDE_API_KEY" ]
    then
        # we want this to throw even if doing a force release because the code checks for the key when in release mode
        echo "error: define the AMPLITUDE_API_KEY environment variable before compiling"
        exit 1
    fi

    if [[ IS_TEST_MODE -eq 0 ]]
    then
        if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc[1-9]+)?$ ]]
        then
            if [[ FORCE_RELEASE -eq 0 ]]
            then
                echo "error: the git commit needs to be tagged with a valid version to prepare archives, see $0 -h for more information"
                exit 1
            else
                echo "force release option set so ignoring version format"
            fi
        fi
        EXPECTED_GIT_RELEASE_BRANCH="release/$(echo $VERSION | cut -d '.' -f1,2).x"
        if ! [[ ("$GIT_BRANCH" == "$EXPECTED_GIT_RELEASE_BRANCH") || ("$GIT_BRANCH" == "master") ]]
        then
            if [[ FORCE_RELEASE -eq 0 ]]
            then
                echo "error: you can only deploy an official release from the 'master' branch or a branch named in the format of 'release/vA.B.x' where 'A' and 'B' are positive numbers that co-incide with the major and minor versions of your release, example: $EXPECTED_GIT_RELEASE_BRANCH"
                exit 1
            else
                echo "force release option set so ignoring release branch requirements"
            fi
        fi
    fi
fi
echo ""

if [[ $ENV == "dev" ]]
then
    GOOS="$(go env GOOS)"
    GOARCH="$(go env GOARCH)"
    echo "GOOS: $GOOS"
    echo "GOARCH: $GOARCH"
    echo ""

    # explicit check for windows
    EXTENSION=""
    if [[ `go env GOOS` == "windows" ]]
    then
        EXTENSION=".exe"
    fi

    # generate outfile
    OUTFILE=bin/kelp$EXTENSION
    mkdir -p bin

    echo -n "compiling ... "
    go build -ldflags "$LDFLAGS" -o $OUTFILE
    check_build_result $?
    echo "successful: $OUTFILE"
    echo ""
    echo "BUILD SUCCESSFUL"
    exit 0
fi
# else, we are in deploy mode

ARCHIVE_DIR=build/$DATE
ARCHIVE_FOLDER_NAME=kelp-$VERSION
ARCHIVE_DIR_SOURCE=$ARCHIVE_DIR/$ARCHIVE_FOLDER_NAME
mkdir -p $ARCHIVE_DIR_SOURCE
OUTFILE=$ARCHIVE_DIR_SOURCE/kelp
cp examples/configs/trader/* $ARCHIVE_DIR_SOURCE/
PLATFORM_ARGS=("darwin amd64" "linux amd64" "windows amd64" "linux arm64" "linux arm 5" "linux arm 6" "linux arm 7")
if [[ IS_TEST_MODE -eq 1 ]]
then
    PLATFORM_ARGS=("$(go env GOOS) $(go env GOARCH)")
fi
for args in "${PLATFORM_ARGS[@]}"
do
    # extract vars
    GOOS=`echo $args | cut -d' ' -f1 | tr -d ' '`
    GOARCH=`echo $args | cut -d' ' -f2 | tr -d ' '`
    GOARM=`echo $args | cut -d' ' -f3 | tr -d ' '`

    # explicit check for windows
    BINARY="$OUTFILE"
    if [[ "$GOOS" == "windows" ]]
    then
        BINARY="$OUTFILE.exe"
    fi

    DYNAMIC_LDFLAGS="$LDFLAGS"
    if [[ "$GOARM" != "" ]]
    then
        GOARM_FLAGS="-X github.com/stellar/kelp/cmd.goarm=$GOARM"
        echo "adding GOARM_FLAGS to DYNAMIC_LDFLAGS: $GOARM_FLAGS"
        DYNAMIC_LDFLAGS="$DYNAMIC_LDFLAGS $GOARM_FLAGS"
    fi

    # compile
    echo -n "compiling for (GOOS=$GOOS, GOARCH=$GOARCH, GOARM=$GOARM) ... "
    env GOOS=$GOOS GOARCH=$GOARCH GOARM=$GOARM go build -ldflags "$DYNAMIC_LDFLAGS" -o $BINARY
    check_build_result $?
    echo "successful"

    # archive
    ARCHIVE_FILENAME=kelp-$VERSION-$GOOS-$GOARCH$GOARM.tar
    cd $ARCHIVE_DIR
    echo -n "archiving binary file ... "
    tar cf ${ARCHIVE_FILENAME} $ARCHIVE_FOLDER_NAME
    TAR_RESULT=$?
    cd $KELP
    if [[ $TAR_RESULT -ne 0 ]]
    then
        echo ""
        echo "archiving failed with error code $TAR_RESULT"
        exit $TAR_RESULT
    fi
    echo "successful: ${ARCHIVE_DIR}/${ARCHIVE_FILENAME}"

    echo -n "cleaning up binary: $BINARY ... "
    rm $BINARY
    echo "successful"
    echo ""
done
echo -n "cleaning up kelp folder: $ARCHIVE_DIR_SOURCE/ ... "
cd $ARCHIVE_DIR
rm $ARCHIVE_FOLDER_NAME/*
rmdir $ARCHIVE_FOLDER_NAME
cd $KELP
echo "done"
echo ""

echo "BUILD SUCCESSFUL"
