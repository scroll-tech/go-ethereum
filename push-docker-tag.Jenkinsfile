credentialDocker = 'dockerhub'

pipeline {
    agent any
    options {
        timeout (20)
    }
    tools {
        go 'go-1.18'
        nodejs "nodejs"
    }
    environment {
        GO111MODULE = 'on'
        PATH="/home/ubuntu/.cargo/bin:$PATH"
        // LOG_DOCKER = 'true'
    }
    stages {
        stage('Tag') {
            steps {
                script {
                    TAGNAME = sh(returnStdout: true, script: 'git tag -l --points-at HEAD')
                    sh "echo ${TAGNAME}"
                    // ... 
                }
            }
        }
        stage('Build') {
            environment {
                // Extract the username and password of our credentials into "DOCKER_CREDENTIALS_USR" and "DOCKER_CREDENTIALS_PSW".
                // (NOTE 1: DOCKER_CREDENTIALS will be set to "your_username:your_password".)
                // The new variables will always be YOUR_VARIABLE_NAME + _USR and _PSW.
                // (NOTE 2: You can't print credentials in the pipeline for security reasons.)
                DOCKER_CREDENTIALS = credentials('dockerhub')
            }
           steps {         
                withCredentials([usernamePassword(credentialsId: "${credentialDocker}", passwordVariable: 'dockerPassword', usernameVariable: 'dockerUser')]) {
                        // Use a scripted pipeline.
                        script {
                            stage('Push image') { 
                                    if (TAGNAME == ""){
                                        return;
                                    }
                                    sh "docker login --username=${dockerUser} --password=${dockerPassword}"
                                    sh "docker build -t scrolltech/l2geth ."
                                    sh "docker tag scrolltech/l2geth:latest scrolltech/l2geth:${TAGNAME}"
                                    sh "docker push scrolltech/l2geth:${TAGNAME}"                
                                }
                        }
                    }
                }
            }
    }
    post {
          success {
            slackSend(message: "l2geth tag ${TAGNAME} build dockersSuccessed")
                catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
                sh '''#!/bin/bash
                      docker rm $(docker ps -a | grep "Exited" | awk '{print $1 }')
                      docker images | grep registry.hub.docker.com/scrolltech/l2geth | awk '{print $3}' | xargs docker rmi -f
                    '''
            }
          }
          // triggered when red sign
          failure {
            slackSend(message: "l2geth tag ${TAGNAME} build docker failed")
          }
          always {
            cleanWs() 
        }
    }
}