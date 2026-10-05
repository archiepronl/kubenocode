import React from 'react';
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';

interface ConfigSidebarProps {
  selectedNode: any;
  schemas: Record<string, any>;
  onUpdate: (nodeId: string, formData: any) => void;
}

export function ConfigSidebar({ selectedNode, schemas, onUpdate }: ConfigSidebarProps) {
  if (!selectedNode) return <div className="sidebar-empty">Select a node to configure</div>;

  // Retrieve the JSON schema for the selected node's connector type
  const nodeSchema = schemas[selectedNode.data.type] || {};

  // Idempotent submit handler: same input data repeatedly applied will yield same DAG state
  const handleSubmit = ({ formData }: any) => {
    onUpdate(selectedNode.id, formData);
  };

  return (
    <div className="sidebar-container" style={{ padding: '20px', background: '#1e1e1e', color: '#fff' }}>
      <h3>Configure: {selectedNode.data.name || selectedNode.id}</h3>
      <Form
        schema={nodeSchema}
        formData={selectedNode.data.params}
        validator={validator}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
