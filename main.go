package main

import (
	"context"
	"fmt"
	"database/sql"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/tidb/store/mockstore"

	"github.com/aws/aws-sdk-go/aws/session"
	_ "github.com/go-sql-driver/mysql"
	tisession "github.com/pingcap/tidb/session"
)

// Local application variables
var (
	awsSession *session.Session
)

type SQLIN struct{
	SQL string `json:"sql"`
}

func HandleRequest(ctx context.Context, event *SQLIN) (string, error) {
	return exec(ctx,event.SQL)
}

const (
	UserName = "root"
	Password = "123456"
	Port     = "4000"
)
var sess tisession.Session

func getDB(ctx context.Context, tidbIP string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/test", UserName, Password, tidbIP, Port)
	return sql.Open("mysql", dsn)
}

func getSession(ctx context.Context) (tisession.Session,error){
	store, err := mockstore.NewMockTikvStore()
	if err != nil {
		return nil,err
	}
	//defer store.Close() //nolint:errcheck
	tisession.SetSchemaLease(0)
	tisession.DisableStats4Test()
	domain, err := tisession.BootstrapSession(store)
	if err != nil {
		return nil,err
	}
	//defer domain.Close()
	domain.SetStatsUpdating(true)
	return tisession.CreateSession4Test(store)
}

func exec(ctx context.Context,sql string) (res string,err error){
	rss,err:=sess.Execute(ctx,sql)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for i,rs:=range rss{
		sb.WriteString(fmt.Sprintf("------RecordSet(%d)------\n",i))
		sRows, err := tisession.ResultSetToStringSlice(ctx, sess, rs)
		if err != nil {
			return "", err
		}
		for _,row:=range sRows{
			for _,c:=range row{
				sb.WriteString(c)
				sb.WriteString("    ")
			}
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func main() {
	mysql.TiDBReleaseVersion="v4.0.9-aws-lambda"
	if sess==nil{
		session,err:=getSession(context.Background())
		if err !=nil{
			panic(err)
		}
		sess=session
	}
	//print(exec(context.Background(),"select tidb_version();"))
	lambda.Start(HandleRequest)
}