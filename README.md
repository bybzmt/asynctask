# asynctask

这是一个给php设计的异步任务调度管理器.

首先需要把需要异步执行的任务写成一个可以访问的地址。

不同的地址之间调度是无序的，同一个地址不同参数的任务是有顺序的。

如果某种任务占用的资源过多，调度器会给这种任务降权，这样保证不会因为某个很慢的任务导至所有的异步任务都卡住。

简单说这个程序所做的就是将url地址从redis中取出然后按一定的优先级依次请求这些url

## 为什么这么做
可以按一般php的写法来做异步任务，不需要考虑cli环境php需要注意的那些问题。如：
* 不需要关心内存泄漏问题
* 不需要关心db/redis等连接超时的问题
* 可以容忍部分php代码有异常，不影响其它正常的代码
* 代码更新即时生效(与一般php一样)
* 运维工作比较简单(只要启动调度器)，其它都是一般的php服务
* 可以直接使用nginx等负载均衡功能！！

## 编译

`go build`

## 使用帮助

```
./asynctask -h
Usage of ./asynctask:
  -addr string
    	listen addr:port (default ":http")
  -baseurl string  #基础url,如果提交的url非完整地址(http开头),自动拼接上它
    	base url (default "")
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
    	redis list key name
  -redis_pwd string
    	redis password

```


## 接口

接口      | 说明
----------| -------
/         | 服务运行状态监控页面
/status   | 监控状态json数据
/task/add | 添加任务接口 参数: method 请求方式, action 任务连接, params 任务参数


## redis接口
可以配置好redis相关设置, 程序会从redis list中获取任务.

任务要求用json格式:`{method:'请求方式', action:'请求地址', 'params':'请求参数字符串'}`

## 注意

* 注意params参数格式为a=1&b=2&c=3。需要使用url编码
* method必需是`GET`或`POST`
* 推荐使用redis来添加任务！

## 监控页截图

![监控页截图](https://raw.githubusercontent.com/bybzmt/asynctask/master/testtools/screenshot.gif)
