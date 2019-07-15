# glog
可跟据指定大小自动拆分日志文件的Golang日志模块

# install
	
	go get github.com/zydp/glog
	
	
# using
    package main

    import (
        "github.com/zydp/glog"
        "sync"
    )

    func main() {
        /* default 100,10 (the logfile file size 100MB, total split 10 times)*/
        logger := glog.New("test.log", "[Info] ", glog.Ldate | glog.Ltime | glog.Lshortfile)
        //logger := glog.NewEx("test.log", "[Info] ", glog.Ldate | glog.Ltime | glog.Lshortfile, 10, 5)
        var Wg sync.WaitGroup

        for i := 0; i < 10; i++ {
            Wg.Add(1)
            go func(count int){
                for i := 0; i < count; i++ {
                    logger.Printf("%s-%d", "abcdefghijklmnopqrstuvwxyz", 123456789)
                    logger.Println("abcdefghijklmnopqrstuvwxyz0123456789你好，我是测试日志~!@#$%^&*()_+{}|:")
                }
                Wg.Done()
            }(100000)
        }
        Wg.Wait()
        logger.Println("It's Done!")
    }
