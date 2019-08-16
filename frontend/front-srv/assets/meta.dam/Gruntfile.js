module.exports = function(grunt) {

    const {Externals} = require('../gui.ajax/res/js/dist/libdefs.js');

    grunt.initConfig({
        babel: {
            options: {
                presets: [
                    ['@babel/preset-env', {
                        targets: "defaults"
                    }],
                    '@babel/preset-react'
                ],
                plugins: [
                    ["@babel/plugin-proposal-decorators", { "legacy": true }],
                    ["@babel/plugin-proposal-class-properties", { "loose" : true }],
                    "@babel/plugin-proposal-function-bind"
                ]
            },

            dist: {
                files: [
                    {
                        expand: true,
                        cwd: 'src/',
                        src: ['**/*.js'],
                        dest: 'build/',
                        ext: '.js'
                    }
                ]
            }
        },
        browserify: {
            ui : {
                options: {
                    external: Externals,
                    browserifyOptions:{
                        standalone: 'PydioDAM',
                        debug: true
                    }
                },
                files: {
                    'build/PydioDAM.js':'build/index.js'
                }
            }
        },
        compress: {
            options: {
                mode: 'gzip',
                level: 9,
            },
            js: {
                expand: true,
                cwd: 'build/',
                src: ['PydioDAM.js'],
                dest: 'build/',
                ext: '.js.gz'
            },
        },
        sass: {
            dist: {
                options: {
                    style: 'expanded'
                },
                files: {
                    'build/PydioDAM.css': 'src/main.scss'
                }
            }
        },
        watch: {
            js: {
                files: [
                    "src/**/*"
                ],
                tasks: ['babel', 'browserify', 'compress', 'sass'],
                options: {
                    spawn: false
                }
            }
        }
    });
    grunt.loadNpmTasks('grunt-browserify');
    grunt.loadNpmTasks('grunt-babel');
    grunt.loadNpmTasks('grunt-contrib-watch');
    grunt.loadNpmTasks('grunt-contrib-compress');
    grunt.loadNpmTasks('grunt-contrib-sass');
    grunt.registerTask('default', ['babel', 'browserify', 'compress', 'sass']);
};
