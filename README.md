# asynctask

这是一个给php设计的异步任务调度管理器.

首先需要把需要异步执行的任务写成一个可以访问的地址. 不同的地扯之间调度器是不保证调用顺序的,但是同一个地址,不同参数的的调用是有顺序的.

调度器会根据地址来调整运行的优先级, 如果某种任务占用的资源过多,调度器会给这种任务降权,这样子保证不会因为某个很慢的任务导至所有的异步任务都卡住.

## 编译

`go build`

## 使用帮助

```
./asynctask -h
Usage of ./asynctask:
  -addr string
    	listen addr:port (default ":http")
  -baseurl string  #异步任务的基础url, 如: http://localhost
    	base url
  -dbfile string  #服务关闭/启动时会将未执行完的任务保存到这个文件中,下次启再时再从中恢复
    	storage file (default "./asynctask.db")
  -max_mem uint  #当服务占用的内存超过这个值时将不再从redis读取新任务了
    	max memory size(MB) (default 128)
  -num int   #同时可以并发执行的请求数量
    	worker number (default 10)
  -redis_db int
    	redis database
  -redis_host string
    	redis host
  -redis_key string
    	redis list key name. json data: {action:string, params:string}
  -redis_pwd string
    	redis password

```


## 接口

接口      | 说明
----------| -------
/         | 服务运行状态监控页面
/status   | 监控状态json数据
/task/add | 添加任务接口 参数: action 任务连接, params 任务参数


## redis接口
可以配置好redis相关设置, 程序会从redis list中获取任务.

任务要求用json格式:`{action:'请求地址', 'params':'请求参数字符串'}`

## 注意

* 注意params参数格式为a=1&b=2&c=3。需要使用url编码
* 任务都必需是POST请求！
* 推荐使用redis来添加任务！

