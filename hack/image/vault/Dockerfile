FROM vault:0.9.1

RUN apk --no-cache add curl
RUN setcap cap_ipc_lock=+ep $(readlink -f $(which vault))
