## :file_folder: The Project
This project is a Lambda function triggered by S3 object creation events. This Lambda processes a csv file containing transactions and send a summary of it by email.

CSV File example:
|Id |Date   |Transaction|
|---|---    |---        |
|0  |7/15   |+60.5      |
|1  |7/28   |-10.3      |
|2  |8/2    |-20.46     |
|3  |8/13   |+10        |

## :rocket: Technologies
|   Back-End       |
| :---:            |
| Go               |
| S3               |
| Lambda function  |
| Serverless       |
| Localstack       |
| Docker-compose   |

## :computer: How to use it
Clone this repository.
___
Set the environment variables in a .env file.
In order to get the mail sending to work, the email must be a gmail account and the "Less secure apps access" option activated.
You can see how to do it here: https://support.google.com/accounts/answer/6010255?hl=en
Then run
```bash
docker-compose up -d --build
```
It will build and run the localstack environment.
___
After that you can run
```bash
make deploy
```
It will deploy the lambda on the localstack environment
___
For testing it you can copy any of the transactions file in the project folder to Localstack docker container. Eg.:
```bash
docker cp transactions2.csv transactions-summary_localstack_1:/opt/code/localstack
```

Then get into Localstack container by running
```bash
docker exec -it transactions-summary_localstack_1 bash
```

And run
```bash
aws --endpoint-url http://localhost:4566 s3 cp transactions2.csv s3://transactions-bucket
```
It will create a new file on the transactions-bucket and trigger the lambda function. You should get an email (at the email set on the .env) with the transactions summary.

## Possible improvements
- Implement mailing with SES
- Trigger lambda by SQS instead of S3
- Validate file format