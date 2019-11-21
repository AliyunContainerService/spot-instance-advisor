# spot-instance-advisor
spot-instance-advisor is command line tool to get the cheapest group of spot instanceTypes.

## Usage 
```$xslt
Usage of ./spot-instance-advisor:
  -accessKeyId string
    	Your accessKeyId of cloud account
  -accessKeySecret string
    	Your accessKeySecret of cloud account
  -cutoff int
    	Discount of the spot instance prices (default 2)
  -family string
    	The spot instance family you want (e.g. ecs.n1,ecs.n2)
  -limit int
    	Limit of the spot instances (default 20)
  -maxcpu int
    	Max cores of spot instances  (default 32)
  -maxmem int
    	Max memory of spot instances (default 64)
  -mincpu int
    	Min cores of spot instances (default 1)
  -minmem int
    	Min memory of spot instances (default 2)
  -region string
    	The region of spot instances (default "cn-hangzhou")
  -resolution int
    	The window of price history analysis (default 7)
```

## Demo 
```$xslt
./spot-instance-advisor --accessKeyId=[id] --accessKeySecret=[secret] --region=cn-zhangjiakou


Initialize cache ready with 619 kinds of instanceTypes
Filter 93 of 98 kinds of instanceTypes.
Fetch 93 kinds of InstanceTypes prices successfully.
Successfully compare 199 kinds of instanceTypes
      InstanceTypeId               ZoneId     Price(Core)        Discount           ratio
        ecs.c6.large     cn-zhangjiakou-c          0.0135             1.0             0.0
        ecs.c6.large     cn-zhangjiakou-a          0.0135             1.0             0.0
      ecs.c6.2xlarge     cn-zhangjiakou-a          0.0136             1.0             0.0
      ecs.c6.2xlarge     cn-zhangjiakou-c          0.0136             1.0             0.0
      ecs.c6.3xlarge     cn-zhangjiakou-a          0.0137             1.0             0.0
      ecs.c6.3xlarge     cn-zhangjiakou-c          0.0137             1.0             0.0
       ecs.c6.xlarge     cn-zhangjiakou-c          0.0138             1.0             0.0
       ecs.c6.xlarge     cn-zhangjiakou-a          0.0138             1.0             0.0
     ecs.hfc6.xlarge     cn-zhangjiakou-a          0.0158             1.0             0.0
      ecs.hfc6.large     cn-zhangjiakou-a          0.0160             1.0             0.0
      ecs.hfc6.large     cn-zhangjiakou-c          0.0160             1.0             0.0
      ecs.g6.3xlarge     cn-zhangjiakou-a          0.0175             1.0             0.0
      ecs.g6.3xlarge     cn-zhangjiakou-c          0.0175             1.0             0.0
        ecs.g6.large     cn-zhangjiakou-a          0.0175             1.0             0.0
       ecs.g6.xlarge     cn-zhangjiakou-a          0.0175             1.0             0.0
      ecs.g6.2xlarge     cn-zhangjiakou-a          0.0175             1.0             0.0
      ecs.g6.2xlarge     cn-zhangjiakou-c          0.0175             1.0             0.0
        ecs.g6.large     cn-zhangjiakou-c          0.0175             1.0             0.0
       ecs.g6.xlarge     cn-zhangjiakou-c          0.0175             1.0             0.0
      ecs.hfg6.large     cn-zhangjiakou-c          0.0195             1.0             0.0
```

## How to create the configure with the result 
* Don't put all the eggs in one bucket
Use 10 kinds of instanceType is a good choice and choose the appropriate weight based on the price.
* Don't choose high ratio instances 
ratio is the standard deviation value of history prices. 