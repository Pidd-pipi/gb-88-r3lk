<template>
  <div ref="editorContainer" class="monaco-editor-container"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, defineProps, defineEmits } from 'vue';
import * as monaco from 'monaco-editor';

const props = defineProps<{
  modelValue: string;
  language?: string;
  readOnly?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const editorContainer = ref<HTMLElement | null>(null);
let editor: monaco.editor.IStandaloneCodeEditor | null = null;

onMounted(() => {
  if (editorContainer.value) {
    editor = monaco.editor.create(editorContainer.value, {
      value: props.modelValue,
      language: props.language || 'json',
      theme: 'vs',
      readOnly: props.readOnly || false,
      minimap: { enabled: false },
      automaticLayout: true,
      fontSize: 14,
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      wordWrap: 'on'
    });

    editor.onDidChangeModelContent(() => {
      if (editor) {
        emit('update:modelValue', editor.getValue());
      }
    });
  }
});

watch(() => props.modelValue, (newValue) => {
  if (editor && editor.getValue() !== newValue) {
    editor.setValue(newValue);
  }
});
</script>

<style scoped>
.monaco-editor-container {
  width: 100%;
  height: 400px;
  border: 1px solid #e5e6eb;
  border-radius: 4px;
}
</style>
