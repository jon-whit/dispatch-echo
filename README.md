# dispatch-demo
This is a demonstration of how a server can dispatch RPCs among peers using gRPC client-side load balancing where
consistent hashing is used to distribute RPCs across peers.

The DispatchEchoService which will dispatch the Echo RPC to its peers based on a consistent hash of the message in the request.

## Quickstart
1. Setup a Kubernetes cluster with `minikube`
```console
➜ minikube start
```

2. Deploy the DispatchEchoService
```console
➜ kubectl apply -f k8s-manifests
```

3. Call the Echo RPC endpoint (using `grpcurl` image)
```console
kubectl run --restart Never --image fullstorydev/grpcurl -- grpcurl -use-reflection -plaintext  -d '{"message": "foo"}' dispatch-echo:50051 dispatch_echo.v1.DispatchEchoService/Echo
```

4. Observe the logs
> ℹ️ What you should see in general is that one of the 3 replicas receives the request, dispatches it to the peer that owns that message, and then that peer serves the Echo RPC.

Looking at the output of the `grpcurl` request, you can see that the Echo RPC was served from the peer whose ip is "10.1.0.56".

```console
➜ kubectl logs -f pod/grpcurl
{
  "message": "foo",
  "peerId": "10.1.0.56"
}
```

Looking at the logs of each individual pod, we can see that pod with ip "10.1.0.58" received the request, dispatched the Echo RPC to it's peer, and that dispatch was received by the pod with ip "10.1.0.56" which served the Echo RPC (via a loopback request to itself):

```console
➜ kubectl logs -f pod/dispatch-echo-89565f69b-ttt9t
2024/07/03 23:45:11 pod ip '10.1.0.58'
2024/07/03 23:45:16 dispatching Echo to peer


➜ kubectl logs -f pod/dispatch-echo-89565f69b-kj8xx
2024/07/03 23:45:10 pod ip '10.1.0.56'
2024/07/03 23:45:16 dispatching Echo to peer
2024/07/03 23:45:16 serving Echo
```
