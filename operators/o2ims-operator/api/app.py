from flask import Flask, request, jsonify
from kubernetes import client, config
from kubernetes.client.rest import ApiException
import os
import logging
from datetime import datetime

app = Flask(__name__)

@app.route('/O2ims_infrastructureProvisioning/v1/provisioningRequests ', methods=['POST'])
def trigger_action():
    data = request.json
    logging.info("O2IMS API Received Request Payload Is:", data)
    # add validation logic here
    
    if data.get('templateName')=='' or data.get('templateVersion')=='' or data.get('templateParameters')=='':
        logging.info("One of the Parameter from templateName,templateVersion,templateParameters is null.. ")
        now = datetime.now()
        dt_string = now.strftime("%Y-%m-%d %H:%M:%S")
        return jsonify({"status":{"updateTime":dt_string,"message":f"O2IMS Deployment Failed,{e}","provisioningPhase":"FAILED"}}),500
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
    try:
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
