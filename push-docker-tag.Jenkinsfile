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
                        def app
                        stage('Docker Build') {
                            if (TAGNAME == ""){
                                    return;
                            }
                            sh 'ls'
                            app = docker.build("${env.DOCKER_CREDENTIALS_USR}/l2geth")
                        }        
                        stage('Push image') { 
                            if (TAGNAME == ""){
                                    return;
                            } 
                            // Use the Credential ID of the Docker Hub Credentials we added to Jenkins.
                            docker.withRegistry('https://registry.hub.docker.com', 'dockerhub') {                                
                                app.push(TAGNAME)
                                // app.push("latest")                      
                            }
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