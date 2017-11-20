pipeline {
    agent {
        docker {
            image 'golang'
        }
    }
    environment { 
        GIT_COMMITTER_NAME = 'jenkins'
        GIT_COMMITTER_EMAIL = 'jenkins@localhost'
        GIT_AUTHOR_NAME = 'jenkins'
        GIT_AUTHOR_EMAIL = 'jenkins@localhost'
        GOPATH = "${WORKSPACE}"
    }
    stages {
        stage('Test') {
            steps {
                sh '''
                    cd ${WORKSPACE}
                    go get github.com/golang/lint/golint
                    go get github.com/tebeka/go2xunit
                    go get git.yale.edu/spinup/reaper/...
                    ./bin/golint src/git.yale.edu/spinup/reaper/.. > lint.txt
                    go test -v $(go list git.yale.edu/spinup/reaper/... | grep -v /vendor/) | ./bin/go2xunit -output tests.xml
                '''
            }
            post {
                success {
                    stash includes: 'lint.txt,test.xml', name: 'reaperTests'
                }
            }
        }
        stage('Build'){
            steps {
                sh 'cd $WORKSPACE && go build -o reaper-native -v git.yale.edu/spinup/reaper'
                sh './reaper-native -version | awk \'{print $3}\'> reaper.version'
                sh '''
                    cd ${WORKSPACE}
                    VERSION=`cat reaper.version`
                    [[ !  -z  ${VERSION}  ]] && echo 'VERSION not found' && exit 1
                    for GOOS in darwin linux; do
                        for GOARCH in 386 amd64; do
                            echo "Building $GOOS-$GOARCH"
                            export GOOS=$GOOS
                            export GOARCH=$GOARCH
                            go build -o reaper-v${VERSION}-$GOOS-$GOARCH git.yale.edu/spinup/reaper
                        done
                    done
                '''
            }
            post {
                success {
                    stash includes: 'reaper*', name: 'reaperBin'
                }
            }
        }
    }
    options {
        buildDiscarder(logRotator(numToKeepStr:'3'))
        timeout(time: 60, unit: 'MINUTES')
    }
}