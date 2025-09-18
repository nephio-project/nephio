from flask import Flask, request, jsonify
from kubernetes import client, config
from kubernetes.client.rest import ApiException
import os
import logging

app = Flask(__name__)

@app.route('/O2ims_infrastructureProvisioning/v1/provisioningRequests ', methods=['POST'])
def trigger_action():
    data = request.json
    logging.info("O2IMS API Received Request Payload Is:", data)
    # add validation logic here
    
    if data.get('templateName')=='' or data.get('templateVersion')=='' or data.get('templateParameters')=='':
        logging.info("One of the Parameter from templateName,templateVersion,templateParameters is null.. ")
        return jsonify({"status":{"updateTime":"","message":f"O2IMS Deployment Failed,{e}","provisioningPhase":"FAILED"}}),500


    o2ims_cr={
            'apiVersion': 'o2ims.provisioning.oran.org/v1alpha1',
            'kind': 'ProvisioningRequest',
            'metadata': {
                'name': data.get('name'),
                'labels':{
                    'provisioningRequestId': data.get('provisioningRequestId')
                }
            },
            'spec':{
                'description': data.get('description'),
                'name':  data.get('name'),
                'templateName': data.get('templateName'),
                'templateParameters':data.get('templateParameters'),
                'templateVersion': data.get('templateVersion')
            }   
    }

    # deploy cr in new thread
    # return response 201 with empty body async
    try:
        #environment = os.getenv("RUNNING_ENVIRONMENT")
        #if environment == "TEST":
        #    config.load_kube_config()
        #else:
        config.load_incluster_config()
        api = client.CustomObjectsApi()
        response = api.create_cluster_custom_object(
                    group='o2ims.provisioning.oran.org',
                    version='v1alpha1',
                    plural='provisioningrequest',
                    body=o2ims_cr
        )
    except client.exceptions.ApiException as e:
        logging.error(f"Caught Exception while deploying O2IMS CR ,{e}")
        return jsonify({"status":{"updateTime":"","message":f"O2IMS Deployment Failed,{e}","provisioningPhase":"FAILED"}}),500
    print(o2ims_cr)
    return jsonify({"provisioningRequestData": data, "status": {"updateTime":"","message":"In-Progress","provisioningPhase":"PROGRESSING"},"ProvisionedResourceSet":{"nodeClusterId":"test","infrastructureResourceIds":"sample"}}), 200
