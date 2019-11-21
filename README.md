# prometheus-lightsail-sd
Service discovery for the [AWS Lightsail](https://aws.amazon.com/lightsail/) platform, compatible with [Prometheus](https://prometheus.io).

## How it works

This service gets the list of servers from the Lightsail API and generates a file which is compatible with the Prometheus `file_sd` mechanism.

## Pre-requisites

### AWS named profile
You will need an AWS named profile config located under ~/.aws/config and ~/.aws/credentials.

The profile name can be specified with either:
* command line argument `--profile=myProfileName`
* setting the environment variable `AWS_PROFILE=myProfileName`

More info: [AWS CLI - Named Profiles](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html)

### IAM policy
The following IAM Policy attached to the profile is required:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "lightsail:GetInstances"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
```

## Installing it

Download the binary from the [Releases](https://github.com/n888/prometheus-lightsail-sd/releases) page.

## Running it

```
usage: prometheus-lightsail-sd usage [<flags>]

Tool to generate file_sd target files for AWS Lightsail.

Flags:
  -h, --help               Show context-sensitive help (also try --help-long and --help-man).
      --output.file="lightsail_sd.json"  
                           Output file for file_sd compatible file.
      --target.refresh=60  The refresh interval (in seconds).
      --profile="default"  AWS config named profile (under ~/.aws, can also be set by envvar: AWS_PROFILE=myProfileName)
```

Example output:

```
$ ./prometheus-lightsail-sd --profile=myProfileName
level=debug ts=2019-11-20T08:56:30.089673118Z caller=main.go:175 msg="loading profile: myProfileName"
level=debug ts=2019-11-20T08:56:30.089986153Z caller=manager.go:224 msg="Starting provider" provider=lightsailSD subs=[lightsailSD]
level=debug ts=2019-11-20T08:56:31.355158911Z caller=main.go:117 msg="get servers" count=2
level=debug ts=2019-11-20T08:56:31.355313346Z caller=main.go:123 msg="server added" source=lightsail/sccbc.net
level=debug ts=2019-11-20T08:56:31.355479273Z caller=main.go:123 msg="server added" source=lightsail/n888.net
```

## Integration with Prometheus

Here is a Prometheus `scrape_config` snippet that configures Prometheus to scrape node_exporter (TCP/9100) on discovered instances:

```yaml
- job_name: node

  # This config assumes that prometheus and prometheus-lightsail-sd are started from the same directory:

  file_sd_configs:
  - files: [ "./lightsail_sd.json" ]

  # The relabeling does the following:
  # - overwrite the scrape address with the node_exporter's port.
  # - add the value of the instance's custom tag named "service".
  # - add the availability zone label.
  # - overwrite the instance label with the server's name.
  
  relabel_configs:
  - source_labels: [__meta_lightsail_private_ip]
    replacement: "${1}:9100"
    target_label: __address__
  - source_labels: [__meta_lightsail_tag_service]
    target_label: service
  - source_labels: [__meta_lightsail_availability_zone]
    target_label: availability_zone
  - source_labels: [__meta_lightsail_name]
    target_label: instance
```

The following meta labels are available on targets during relabeling:

* `__meta_lightsail_availability_zone="us-west-2a"`: availability zone
* `__meta_lightsail_blueprint_id="ubuntu_18_04"`: blueprint id (lightsail pre-baked image)
* `__meta_lightsail_bundle_id="nano_2_0"`: instance size
* `__meta_lightsail_instance_id="i-00f8cb1818387aa7z"`: instance id
* `__meta_lightsail_name="n888.net"`: instance name
* `__meta_lightsail_private_ip="172.88.3.6"` instance private ip
* `__meta_lightsail_public_ip="50.33.32.72"` instance public ip
* `__meta_lightsail_state="running"` instance state
* `__meta_lightsail_support_code="782236961567/i-00f8cb1818387aa7z"` instance support code
* `__meta_lightsail_tag_service="frontend"` instance tag, each tag gets its own label

```
note: lightsail.Instance does not return an InstanceId (unlike EC2), but does provide a 
"support code" string with the format of "${lightsail_account_id}/${instance_id}". This
string is split and the ${instance_id} value is assigned to `instance_id` tag label.
```

## Contributing

Pull requests, issues and suggestions are appreciated.

## Credits

* Prometheus Authors 
  * https://prometheus.io  
  * https://prometheus.io/blog/2018/07/05/implementing-custom-sd  
  * https://github.com/prometheus/prometheus/tree/master/discovery/ec2  
* Scaleway Service Discovery
  * Core code based on https://github.com/scaleway/prometheus-scw-sd  
* aws-sdk-go
  * https://aws.amazon.com/sdk-for-go  

## License

Apache License 2.0, see [LICENSE](https://github.com/n888/prometheus-lightsail-sd/blob/master/LICENSE).
