docker build --pull --tag localhost/ipcoin .
rm -rf vendor
docker save localhost/ipcoin | ssh -C va1 sudo docker load
