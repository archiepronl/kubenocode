import React from 'react';
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';

export function ConfigSidebar({ selectedNode, schemas, onUpdate }) {
  if (!selectedNode) return <div style={{ padding: '20px' }}>Select a node to configure</div>;

  const nodeSchema = schemas[selectedNode.data.type] || {};

  const handleSubmit = ({ formData }) => {
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
        className="rjsf-dark"
      />
    </div>
  );
}
