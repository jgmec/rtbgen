# Write golang cli pplication for generating random rtb 2.5 requests with the following requirements:  
- configuration should be loaded on startup from cli parameters
- rtb requests should be generated for a period of current timestamp minus 5 minutes
- number of rtb requests should be configurable via cli param count
- optionally rtb requests should be generated with locations in bounding box ( max lat , max lon, min lat, min lon )
# application should support also http service for staring of long running tasks, payload for task create handler should accept the following parameteres:
- start time
- end time
- criteria - one of IP address, IFA, bounding box ( max lat , max lon, min lat, min lon )
- task correlation id
- number of rtb requests
- application should persists all task requests
- application should generate each five minutes rtb requests for all running tasks ( all tasks for whichh current timestamp is between start time and end time) and save them to a file as json lines
- application should add files generated in last run into a zip file and remove jsonl files
- application should upload zip file to specified sftp server and delete local zip file

# If you have any questions please ask