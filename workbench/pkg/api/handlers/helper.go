package handlers

import (
	"testing"

	"github.com/emicklei/go-restful/v3"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gokube/pkg/storage"
)

func withTestServer(t *testing.T, callback func(etcdServer *clientv3.Client, ws *restful.WebService, container *restful.Container)) {
	storage.TestWithEmbeddedEtcd(t, func(t *testing.T, etcdServer *clientv3.Client) {
		container := restful.NewContainer()
		ws := new(restful.WebService)

		ws.Path("/api/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
		container.Add(ws)

		callback(etcdServer, ws, container)
	})
}
