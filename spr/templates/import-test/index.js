// Test import/initialization behavior of target package
// This script is executed after npm install to trigger init-time attacks

console.log('=== TEST: Import/Initialization ===');
console.log('Target: {{.PackageName}}@{{.PackageVersion}}');
console.log('Module type: {{.ModuleType}}');
console.log('');

{{if eq .ModuleType "module"}}
// ESM import
import * as target from '{{.PackageName}}';

console.log('');
console.log('=== Import successful ===');
console.log('Exports:', Object.keys(target).slice(0, 20));
console.log('Export count:', Object.keys(target).length);
{{else}}
// CommonJS require
const target = require('{{.PackageName}}');

console.log('');
console.log('=== Import successful ===');
console.log('Exports:', Object.keys(target).slice(0, 20));
console.log('Export count:', Object.keys(target).length);
{{end}}
