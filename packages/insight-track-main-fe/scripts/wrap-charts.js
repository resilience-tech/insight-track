import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

// The directory containing the components
const componentsDir = '../src/charts'; // Change this to your components directory

// Function to wrap component's content with ChartContainer
const wrapWithChartContainer = (content) => {
  const importMetaUrlLine = `const filepath = new URL(import.meta.url).pathname;`;

  // Match all import statements
  const importStatements = content.match(/import[\s\S]*?from\s+['"][\s\S]*?['"];?\n/g) || [];

  // Remove import statements from the original content
  const contentWithoutImports = content.replace(/import[\s\S]*?from\s+['"][\s\S]*?['"];?\n/g, '').trim();

  // This regular expression matches the default export component function
  const componentFunctionRegex = /const\s+(\w+)\s*=\s*\(\)\s*=>\s*{\s*return\s*\(([\s\S]*?)\);\s*}/;
  const match = contentWithoutImports.match(componentFunctionRegex);
  
  if (match) {
    const componentName = match[1];
    const returnContent = match[2].trim();

    // Combine original import statements with the new import
    const combinedImports = importStatements.join('') + `import ChartContainer from '../../LayoutComponents/ChartContainer';\n`;

    return `
${combinedImports}

${importMetaUrlLine}

const ${componentName} = () => {
  return (
    <ChartContainer filepath={filepath}>
      ${returnContent}
    </ChartContainer>
  );
}

export default ${componentName};
`;
  } else {
    return content;
  }
};

// Function to process each file in the directory
const processFile = (filePath) => {
  fs.readFile(filePath, 'utf-8', (err, data) => {
    if (err) {
      console.error(`Error reading file ${filePath}:`, err);
      return;
    }

    const wrappedContent = wrapWithChartContainer(data);

    fs.writeFile(filePath, wrappedContent, 'utf-8', (err) => {
      if (err) {
        console.error(`Error writing file ${filePath}:`, err);
      } else {
        console.log(`Successfully wrapped ${filePath}`);
      }
    });
  });
};

// Function to traverse the directory and process each file
const traverseDirectory = (dir) => {
  fs.readdir(dir, (err, files) => {
    if (err) {
      console.error(`Error reading directory ${dir}:`, err);
      return;
    }

    files.forEach((file) => {
      const filePath = path.join(dir, file);
      fs.stat(filePath, (err, stat) => {
        if (err) {
          console.error(`Error stating file ${filePath}:`, err);
          return;
        }

        if (stat.isFile() && file.endsWith('.tsx')) {
          processFile(filePath);
        } else if (stat.isDirectory()) {
          traverseDirectory(filePath);
        }
      });
    });
  });
};

// Start processing the directory
traverseDirectory(componentsDir);
