image_id=$1
service=$2

target_tag=target.production

docker pull 810028259040.dkr.ecr.us-east-1.amazonaws.com/passage:$image_id
docker tag 810028259040.dkr.ecr.us-east-1.amazonaws.com/passage:$image_id 810028259040.dkr.ecr.us-east-1.amazonaws.com/passage:$target_tag
docker push 810028259040.dkr.ecr.us-east-1.amazonaws.com/passage:$target_tag
#aws ecs update-service --cluster passage_$service --service passage_$service --force-new-deployment
