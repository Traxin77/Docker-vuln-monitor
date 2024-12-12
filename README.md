
# Docker-vuln monitor
docker-vuln monitor is an opensource project which automatically vulnerability scans the github repositories which can be dockerized which is an extension to the docker-vuln project where when we use docker-vuln cli tool with monitor flag we are able to send the github repo links to this 24/7 running server which montors those links and whenever there is an push event on the gtihub repository docker-vuln monitor automatically scans the repository and stores its results on mongodb 




## Deployment

To deploy this project run

```bash
  cd docker-vuln monitor
  docker-compose build
  docker-compose up
```


## Authors

- [@Traxin](https://github.com/Traxin77)

