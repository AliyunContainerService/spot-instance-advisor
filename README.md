# spot-instance-advisor
spot-instance-advisor is a server which is able to get the cheapest group and send alarm of spot instanceTypes depend on your request(config).Spot-instance-advisor takes DingDing groups as default message receivers.

## Build binary    
```$xslt
make bin 
```

## Usage 
```$xslt
Usage of ./spot-instance-advisor
```
## Config
spot-instance-advisor reads config from env variable. Some of config is mandatory, while the others would be given default values if you choose not to config. If you want to get alarm of spot instanceTypes, you should pay more attention to the ```alarm_config```, which would be given more details in Demo. 

### Explanations
| env name | mandatory | meaning | format | demo | default value |
| :--- | :--- | :--- | :--- | :--- | :--- |
| access_key_id | Y | aliyun access_key_id  | string |  |  |
| access_key_secret | Y | aliyun access_key_secret | string |  |  |
| sender_tokens | Y |  | json | sender_tokens='{"group_one":"token_one","group_two":"token_two"}' | as same as demo |
| sender | N |  | json | sende='{"sender_name":"DingDing","threshold":"0.1"}' | as same as demo |
| filter | N |  | json | filter='{"filter":"price_per_core","threshold":"0.1"}' | as same as demo |
| message_pattern | N |  | json | pattern='{"color":true,"color_field":"price_per_core"}' | as same as demo |
| log_level | N |  | string and must be one ofbelows debug/info/warning/error/fatal/panic | log_level='warning' | 
as same as demo |
| not_empty_field | N |  | json | not_empty_field="["sys.ding.conversationTitle","region"]" | as same as demo |
| alarm_config | depends on whether alarm function is needed |  | json | see demo |  |

## Demo 
### Config Demo
```
export sender_tokens='{"group_one":"group_one_token","group_two":"group_one_two"}'
export access_key_id='XXXXXX'
export access_key_secret='XXXXXX'
export log_level='info'
export alarm_config='{"group_one":[{"cron":"* * * * *","cons_str":"{\"region\": \"cn-beijing\"}","cons":{"region":"北京"},"sender":[{"sender_name":"DingDing","token":{"group_one":"group_one robot token"}}],"filter":{"judged_by_field":"price_per_core","threshold":"0.123"},"pattern":{"color":true,"color_field":"InstanceTypeId"},"not_empty_field":["sys.ding.conversationTitle","region"]}]}'
```

Let's explain the ```alarm_config```config details.
* ```alarm_config``` must be a json format string. 
* It configs in group. ```group_one``` is a DingDing group's name, and ```group_two``` is the other DingDing group's name.
* You can set one or more query criteria in ```cons_str```, and make sure it can be decoded as json
* Spot instances whose value are bigger than the ```threshold``` would be alarmed.

### use cases
#### query case
```
http://local.elenet.me:8000/spot?region=cn-beijing&&sys.ding.conversationTitle=group_one
```

```
group_one gets result like this:
查询结果
机型       	   可用区   核时价格
sn1ne.3xlarge   f      0.03392
sn1ne.4xlarge	f	   0.04394
sn1ne.2xlarge	f	   0.05400
```

#### alarm case
if you config the alarm_config, the group would get the alarm result period
```
监控提醒
机型       	   可用区   核时价格
sn1ne.3xlarge   f      0.03392
sn1ne.4xlarge	f	   0.04394
sn1ne.2xlarge	f	   0.05400
```


## How to create the configure with the result 
* Don't put all the eggs in one bucket
Use 10 kinds of instanceType is a good choice and choose the appropriate weight based on the price.
* Don't choose high ratio instances 
ratio is the standard deviation value of history prices. 