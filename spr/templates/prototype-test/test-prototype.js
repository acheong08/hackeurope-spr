// Test for prototype pollution behavior
// Records baseline, imports package, triggers any polluted properties

console.log('=== TEST: Prototype Pollution Detection ===');
console.log('Target: {{.PackageName}}@{{.PackageVersion}}');
console.log('');

// Record baseline state of Object.prototype
const baselineProps = Object.getOwnPropertyNames(Object.prototype);
console.log('Baseline Object.prototype property count:', baselineProps.length);
console.log('');

{{if eq .ModuleType "module"}}
// ESM - need async wrapper
(async () => {
  // Import the target package
  const target = await import('{{.PackageName}}');
  
  console.log('');
  console.log('Package imported successfully');
  
  // Check for changes to Object.prototype
  const afterProps = Object.getOwnPropertyNames(Object.prototype);
  const addedProps = afterProps.filter(prop => !baselineProps.includes(prop));
  
  console.log('');
  console.log('=== Prototype Analysis ===');
  console.log('Object.prototype property count after import:', afterProps.length);
  console.log('New properties added:', addedProps.length);
  
  if (addedProps.length > 0) {
    console.log('');
    console.log('New properties detected:', addedProps);
    console.log('');
    console.log('Triggering new properties...');
    
    for (const prop of addedProps) {
      try {
        const value = Object.prototype[prop];
        console.log(`  Property: ${prop}, Type: ${typeof value}`);
        
        if (typeof value === 'function') {
          console.log(`  -> Calling ${prop}()`);
          value();
          console.log(`  -> ${prop}() executed`);
        } else {
          console.log(`  -> Value:`, value);
        }
      } catch (err) {
        console.log(`  -> Error accessing ${prop}:`, err.message);
      }
    }
  }
  
  console.log('');
  console.log('=== Test complete ===');
})();
{{else}}
// CommonJS
// Import the target package
const target = require('{{.PackageName}}');

console.log('');
console.log('Package imported successfully');

// Check for changes to Object.prototype
const afterProps = Object.getOwnPropertyNames(Object.prototype);
const addedProps = afterProps.filter(prop => !baselineProps.includes(prop));

console.log('');
console.log('=== Prototype Analysis ===');
console.log('Object.prototype property count after import:', afterProps.length);
console.log('New properties added:', addedProps.length);

if (addedProps.length > 0) {
  console.log('');
  console.log('New properties detected:', addedProps);
  console.log('');
  console.log('Triggering new properties...');
  
  for (const prop of addedProps) {
    try {
      const value = Object.prototype[prop];
      console.log(`  Property: ${prop}, Type: ${typeof value}`);
      
      if (typeof value === 'function') {
        console.log(`  -> Calling ${prop}()`);
        value();
        console.log(`  -> ${prop}() executed`);
      } else {
        console.log(`  -> Value:`, value);
      }
    } catch (err) {
      console.log(`  -> Error accessing ${prop}:`, err.message);
    }
  }
}

console.log('');
console.log('=== Test complete ===');
{{end}}
