pipeline {
    agent {
        docker {
            image 'golang:1.11'
        }
    }
    environment { 
        GIT_COMMITTER_NAME = 'jenkins'
        GIT_COMMITTER_EMAIL = 'jenkins@localhost'
        GIT_AUTHOR_NAME = 'jenkins'
        GIT_AUTHOR_EMAIL = 'jenkins@localhost'
        GOPATH = "/go"
        GOCACHE="/tmp/.cache"
    }
    stages {
        stage('Test') {
            steps {
                sh '''
                    cd /go && go get -u golang.org/x/lint/golint
                    cd /go && go get -u github.com/tebeka/go2xunit
                    cd ${WORKSPACE}
                    go get ./...
                    /go/bin/golint ./.. > lint.txt
                    go test -v $(go list ./... | grep -v /vendor/) | /go/bin/go2xunit -output tests.xml
                '''
            }
            post {
                success {
                    stash includes: 'lint.txt,test.xml', name: 'reaperTests'
                }
            }
        }
    }
    options {
        buildDiscarder(logRotator(numToKeepStr:'3'))
        timeout(time: 60, unit: 'MINUTES')
    }
}